package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/pheelee/traefik-admin/config"
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

	// Create sys configs
	mw := config.Config{
		HTTP: config.HTTP{
			Middlewares: make(map[string]*config.Middleware),
		},
	}
	mw.HTTP.Middlewares["sys-redirscheme"] = &config.Middleware{
		RedirectScheme: config.RedirectScheme{
			Scheme:    "https",
			Permanent: true,
		},
	}
	mw.HTTP.Middlewares["sys-hsts"] = &config.Middleware{
		Headers: config.Headers{
			STSSeconds: 31536000,
		},
	}

	if err := mw.Write(path.Join(cfg.ConfigPath, "sys_middlewares.yaml")); err != nil {
		panic(err)
	}

	r := SetupRoutes(cfg)
	logger.Info(fmt.Sprintf("Starting server on :%d", port))
	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
