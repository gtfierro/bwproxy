package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type proxyServer struct {
	port       string
	staticpath string
	router     *httprouter.Router
	registry   *registry
}

func startProxyServer(cfg *Config) {
	server := &proxyServer{
		port:       cfg.Port,
		staticpath: cfg.StaticPath,
	}
	server.router = httprouter.New()

	registryPath := cfg.StaticPath + "/.registry.db"
	server.registry = newRegistry(registryPath, cfg.BOSSWAVEAgent)

	server.router.POST("/streaming", server.doStreamingCall)
	server.router.POST("/call", server.doCall)
	//r.ServeFiles("/static/*filepath", http.Dir(cfg.StaticPath+"/static"))
	// TODO: think about how to "install" apps. Do we just place the source in a known folder?
	// TODO: need a way to "isolate" apps: chroot? https://github.com/adtac/fssb? Docker?

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

	permissions, err := srv.registry.getPermissions(rpc_params.Key)
	if err != nil {
		log.Error(err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}

	log.Debug(permissions)
}

// get the key from the request, fetch the permissions from the registry
func (srv *proxyServer) doCall(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rpc_params BWRPCCall

	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

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
