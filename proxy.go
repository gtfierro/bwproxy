package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

var upgrader = websocket.Upgrader{} // default
var badPathMatch = regexp.MustCompile("^(/|\\.)")

type proxyServer struct {
	port          string
	useipv6       bool
	listenaddress string
	staticpath    string
	// path where applications will be installed
	apppath string

	// app configuration
	runningApps    map[string]*appServer
	portRangeStart int
	usedPorts      map[string]int
	portLock       sync.Mutex

	router   *httprouter.Router
	registry *registry
}

func startProxyServer(cfg *Config) {
	server := &proxyServer{
		port:           cfg.Port,
		useipv6:        cfg.UseIPv6,
		listenaddress:  cfg.ListenAddress,
		staticpath:     cfg.StaticPath + "/static",
		apppath:        cfg.AppPath,
		runningApps:    make(map[string]*appServer),
		portRangeStart: cfg.PortRangeStart,
		usedPorts:      make(map[string]int),
	}
	server.router = httprouter.New()

	registryPath := cfg.StaticPath + "/.registry.db"
	server.registry = newRegistry(registryPath, cfg.BOSSWAVEAgent)

	server.router.ServeFiles("/static/*filepath", http.Dir(server.staticpath))

	// BW2 API calls
	server.router.GET("/streaming", server.doStreamingCall)
	server.router.POST("/call", server.doCall)

	// app browsing/managemenet
	server.router.GET("/apps/list", server.listApps)
	server.router.GET("/apps/start/:name", server.startApp)

	server.router.GET("/", server.phoneHome)
	server.router.GET("/browse", server.browse)
	// TODO: think about how to "install" apps. Do we just place the source in a known folder?
	// TODO: need a way to "isolate" apps: chroot? https://github.com/adtac/fssb? Docker?
	// TODO: need a way to prevent apps from calling "across" each other

	// configure server
	var (
		addrString string
		nettype    string
	)

	// check if ipv6
	if cfg.UseIPv6 {
		nettype = "tcp6"
		addrString = "[" + cfg.ListenAddress + "]:" + server.port
	} else {
		nettype = "tcp4"
		addrString = cfg.ListenAddress + ":" + server.port
	}

	address, err := net.ResolveTCPAddr(nettype, addrString)
	if err != nil {
		log.Fatalf("Error resolving address %s (%s)", addrString, err.Error())
	}

	http.Handle("/", server.router)
	log.Notice("Starting HTTP Server on ", addrString)

	srv := &http.Server{
		Addr: address.String(),
	}
	log.Fatal(srv.ListenAndServe())
}

// get the key from the request, fetch the permissions from the registry
func (srv *proxyServer) doStreamingCall(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rpc_params BWRPCCall

	ctx := req.Context()

	c, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Error(err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}
	defer c.Close()
	for {
		// fetch the RPC params
		if err := c.ReadJSON(&rpc_params); err != nil {
			log.Error(err)
			rw.WriteHeader(400)
			rw.Write([]byte(err.Error()))
			return
		}

		if rpc_params.Key == "" {
			log.Error("Empty api key!")
			rw.WriteHeader(400)
			rw.Write([]byte("Empty API key in request"))
			return
		}

		permissions, err := srv.registry.getPermissions(rpc_params.Key)
		if err != nil {
			log.Error(err)
			rw.WriteHeader(500)
			rw.Write([]byte(err.Error()))
			return
		}

		log.Debugf("%+v", permissions)
		log.Debugf("%+v", rpc_params)

		// get the client for the vk
		client := srv.registry.getClientForVK(permissions.VK)
		if client == nil {
			log.Error("No associated client for that VK")
			rw.WriteHeader(500)
			rw.Write([]byte("No associated client for that VK"))
			return
		}

		respchan, errchan := doRPCStream(ctx, client, permissions, rpc_params)
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Error(err)
				rw.WriteHeader(500)
				rw.Write([]byte(err.Error()))
				return
			case err := <-errchan:
				log.Error(err)
				rw.WriteHeader(500)
				rw.Write([]byte(err.Error()))
				return
			case resp := <-respchan:
				if err := c.WriteMessage(websocket.TextMessage, resp); err != nil {
					log.Error(err)
					rw.WriteHeader(500)
					rw.Write([]byte(err.Error()))
					return
				}
			}
		}
	}
}

// get the key from the request, fetch the permissions from the registry
func (srv *proxyServer) doCall(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rpc_params BWRPCCall

	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	rw.Header().Set("Content-Type", "application/json")

	defer req.Body.Close()
	dec := json.NewDecoder(req.Body)

	// fetch the RPC params
	if err := dec.Decode(&rpc_params); err != nil {
		log.Error(err)
		rw.WriteHeader(400)
		rw.Write([]byte(err.Error()))
		return
	}

	if rpc_params.Key == "" {
		log.Error("Empty api key!")
		rw.WriteHeader(400)
		rw.Write([]byte("Empty API key in request"))
		return
	}

	// get permissions for the key (and the vk)
	permissions, err := srv.registry.getPermissions(rpc_params.Key)
	if err != nil {
		log.Error(err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}

	log.Debugf("%+v", permissions)
	log.Debugf("%+v", rpc_params)

	// get the client for the vk
	client := srv.registry.getClientForVK(permissions.VK)
	if client == nil {
		log.Error("No associated client for that VK")
		rw.WriteHeader(500)
		rw.Write([]byte("No associated client for that VK"))
		return
	}

	// do the call and get the results
	results, err := doRPCCall(ctx, client, permissions, rpc_params)
	if err != nil {
		log.Error(err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.Write(results)
	return
}

func (srv *proxyServer) phoneHome(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	defer req.Body.Close()
	log.Notice("Serving", srv.staticpath+"/home.html", "to", req.RemoteAddr)
	http.ServeFile(rw, req, srv.staticpath+"/home.html")
}

func (srv *proxyServer) browse(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	defer req.Body.Close()
	http.ServeFile(rw, req, srv.staticpath+"/browse.html")
}

//TODO: if already started, attach the port number. If not, then render the app start
func (srv *proxyServer) listApps(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	defer req.Body.Close()

	var manifests []appManifest

	// list apps
	appManifests, _ := filepath.Glob(srv.apppath + "/*/manifest.json")
	for _, manifestpath := range appManifests {
		file, err := os.Open(manifestpath)
		if err != nil {
			log.Error(errors.Wrapf(err, "Could not load manifest %s", manifestpath))
			rw.WriteHeader(500)
			rw.Write([]byte(err.Error()))
			return
		}
		var manifest appManifest
		if err := json.NewDecoder(file).Decode(&manifest); err != nil {
			log.Error(errors.Wrapf(err, "Could not decode manifest %s", manifestpath))
			rw.WriteHeader(500)
			rw.Write([]byte(err.Error()))
			return
		}
		if app, found := srv.runningApps[manifest.Name]; found {
			manifest.Address = srv.listenaddress + ":" + app.port
		} else {
			manifest.Address = srv.listenaddress + ":" + srv.port + "/apps/start/" + manifest.Name
		}
		manifests = append(manifests, manifest)
	}

	err := json.NewEncoder(rw).Encode(manifests)
	if err != nil {
		log.Error(errors.Wrap(err, "Could not write manifest response"))
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}
	return
}

func (srv *proxyServer) startApp(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	defer req.Body.Close()

	appname := ps.ByName("name")
	if appname == "" || badPathMatch.MatchString(appname) {
		err := "Could not open app with invalid name " + appname
		log.Error(err)
		rw.WriteHeader(500)
		rw.Write([]byte(err))
		return
	}
	file, err := os.Open(srv.apppath + "/" + appname + "/manifest.json")
	if err != nil {
		err = errors.Wrapf(err, "Could not open manifest for app %s", appname)
		log.Error(err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}
	var manifest appManifest
	if err := json.NewDecoder(file).Decode(&manifest); err != nil {
		log.Error(errors.Wrapf(err, "Could not decode manifest %s", srv.apppath+"/"+appname+"/manifest.json"))
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}

	cfg := &appConfig{
		port:          srv.getFreePort(manifest.Name),
		useipv6:       srv.useipv6,
		listenaddress: srv.listenaddress,
		root:          srv.apppath + "/" + appname,
		proxy:         srv,
	}
	log.Notice("Starting", manifest, "on", cfg.port)
	log.Noticef("%+v", cfg)
	app := startAppServer(cfg)
	srv.runningApps[appname] = app

	// now redirect to the running app
	http.Redirect(rw, req, "http://"+cfg.listenaddress+":"+cfg.port, http.StatusFound)
	return
}

// gets an open port number for the given application
func (srv *proxyServer) getFreePort(name string) string {
	srv.portLock.Lock()
	defer srv.portLock.Unlock()
	var newport int
	for newport = srv.portRangeStart; newport-srv.portRangeStart < 100; newport++ {
		for _, port := range srv.usedPorts {
			if port == newport {
				break
			}
		}
		break
	}
	srv.usedPorts[name] = newport
	log.Debug(newport, srv.portRangeStart)
	return strconv.Itoa(newport)
}
