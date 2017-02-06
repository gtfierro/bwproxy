package main

import (
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type appServer struct {
	running bool

	port string
	// filesystem path where the app is located
	root string
	// permission key
	key string

	// router
	router *httprouter.Router
	// for proxy server calls
	proxy *proxyServer
}

type appConfig struct {
	port          string
	useipv6       bool
	listenaddress string
	root          string
	proxy         *proxyServer
}

type appManifest struct {
	Name        string
	Description string
	Version     string
	// not going to be populated by the manifest
	Address string
}

func startAppServer(cfg *appConfig) *appServer {
	app := &appServer{
		running: false,
		port:    cfg.port,
		root:    cfg.root,
		proxy:   cfg.proxy,
	}
	app.router = httprouter.New()
	log.Debug(app.root)
	log.Debugf("%+v", http.Dir(app.root))
	app.router.ServeFiles("/static/*filepath", http.Dir(app.root))

	// serve app's index.html
	app.router.GET("/", app.index)

	// pass through
	app.router.GET("/streaming", app.proxy.doStreamingCall)
	app.router.POST("/call", app.proxy.doCall)
	// serve the bw2lib.js file
	app.router.GET("/js/bw2lib.js", app.serveJS)

	// configure server
	var (
		addrString string
		nettype    string
	)

	// check if ipv6
	if cfg.useipv6 {
		nettype = "tcp6"
		addrString = "[" + cfg.listenaddress + "]:" + cfg.port
	} else {
		nettype = "tcp4"
		addrString = cfg.listenaddress + ":" + cfg.port
	}

	address, err := net.ResolveTCPAddr(nettype, addrString)
	if err != nil {
		log.Fatalf("Error resolving address %s (%s)", addrString, err.Error())
	}

	log.Notice("Starting HTTP Server on ", addrString)

	go func() {
		http.ListenAndServe(address.String(), app.router)
	}()

	app.running = true
	return app
}

func (app *appServer) stop() {
	app.running = false
	// TODO: wait for golang 1.8 graceful shutdown
}

func (app *appServer) index(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	defer req.Body.Close()
	http.ServeFile(rw, req, app.root+"/index.html")
}

func (app *appServer) serveJS(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	defer req.Body.Close()
	log.Debug("here", app.proxy.staticpath+"/js/bw2lib.js")
	http.ServeFile(rw, req, app.proxy.staticpath+"/js/bw2lib.js")

}
