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
	ConfigPath    string
	WebRoot       string
	ConfigOptions config.Options
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var port int
	cfg := appConfig{}

	flag.StringVar(&cfg.ConfigPath, "ConfigPath", "", "path where the dynamic config files getting stored")
	flag.StringVar(&cfg.WebRoot, "WebRoot", "", "defines the WebRoot containing index.html and static resources")
	flag.StringVar(&cfg.ConfigOptions.CertResolver, "CertResolver", "", "name of the cert resolver which is configured for traefik, e.g http01 or dns01")
	flag.IntVar(&port, "Port", 8099, "Listening Port")

	flag.Parse()

	if cfg.ConfigPath == "" || cfg.WebRoot == "" || cfg.ConfigOptions.CertResolver == "" {
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

	// Migrate certResolver for all configs to the specified one
	names, err := config.List(cfg.ConfigPath)
	check(err)
	for _, n := range names {
		path := path.Join(cfg.ConfigPath, n+".yaml")
		c, err := config.Get(path)
		check(err)
		if c.HTTP.Routers[n].TLS != nil {
			c.HTTP.Routers[n].TLS.CertResolver = cfg.ConfigOptions.CertResolver
			check(c.Write(path))
		}
	}

	r := SetupRoutes(cfg)
	logger.Info(fmt.Sprintf("Starting server on :%d", port))
	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
