package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/pheelee/traefik-admin/logger"
)

type appConfig struct {
	ConfigPath string
	WebRoot    string
}

func main() {
	var port int
	cfg := appConfig{
		ConfigPath: os.Getenv("CONFIG_PATH"),
	}

	flag.StringVar(&cfg.ConfigPath, "ConfigPath", "", "path where the dynamic config files getting stored")
	flag.StringVar(&cfg.WebRoot, "WebRoot", "", "defines the WebRoot containing index.html and static resources")
	flag.IntVar(&port, "Port", 8099, "Listening Port")

	flag.Parse()

	if cfg.ConfigPath == "" || cfg.WebRoot == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	r := SetupRoutes(cfg)
	logger.Info(fmt.Sprintf("Starting server on :%d", port))
	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
