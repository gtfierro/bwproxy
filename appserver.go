package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type appServer struct {
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
	port    string
	useipv6 bool
	root    string
}

func (app *appServer) start(cfg *appConfig) {
	app.router = httprouter.New()
	app.router.ServeFiles("/", http.Dir(app.root))

	// serve app's index.html
	app.router.GET("/", app.index)

	// pass through
	app.router.GET("/streaming", app.proxy.doStreamingCall)
	app.router.POST("/call", app.proxy.doCall)
	// serve the bw2lib.js file
	app.router.ServeFiles("bw2lib.js", http.Dir(app.proxy.staticpath+"/static/js"))

	//TODO: need to get the port we're going to serve on
	// want a hostname as well?
}

func (app *appServer) stop() {
}

func (app *appServer) index(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
}
