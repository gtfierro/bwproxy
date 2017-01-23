package main

import (
	"os"

	"github.com/op/go-logging"
	"github.com/urfave/cli"
)

// logger
var log *logging.Logger

// set up logging facilities
func init() {
	log = logging.MustGetLogger("bwproxy")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type Config struct {
	Port          string
	ListenAddress string
	StaticPath    string
	UseIPv6       bool
	BOSSWAVEAgent string
}

func main() {
	app := cli.NewApp()
	app.Name = "bwproxy"
	app.Version = "0.1.0"
	app.Usage = "BOSSWAVE HTTP Proxy for sandboxed applications"

	app.Commands = []cli.Command{
		{
			Name:   "register",
			Usage:  "Register a new API key",
			Action: doRegister,
		},
		{
			Name:   "proxy",
			Usage:  "Run the proxy",
			Action: runProxy,
		},
	}
	app.Run(os.Args)
}
