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
	ConfigPath            string
	WebRoot               string
	AuthorizationEndpoint string
	CookieSecret          string
	ConfigOptions         config.Options
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
	flag.StringVar(&cfg.WebRoot, "WebRoot", "", "defines the WebRoot containing index.html and static resources (for development)")
	flag.StringVar(&cfg.ConfigOptions.CertResolver, "CertResolver", "", "name of the cert resolver which is configured for traefik, e.g http01 or dns01")
	flag.StringVar(&cfg.AuthorizationEndpoint, "AuthEndpoint", "", "indieauth authorization endpoint for auth forwarding, e.g https://homeassistant.tld/auth/authorize")
	flag.StringVar(&cfg.CookieSecret, "CookieSecret", "", "secret to encode session cookie (use strong random string)")
	flag.IntVar(&port, "Port", 8099, "Listening Port")

	flag.Parse()

	if cfg.ConfigPath == "" || cfg.ConfigOptions.CertResolver == "" || cfg.CookieSecret == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	// Create sys configs
	mw := config.Config{
		HTTP: config.HTTP{
			Middlewares: make(map[string]*config.Middleware),
		},
	}

	mw.HTTP.Middlewares[config.REDIRSCHEME] = &config.Middleware{
		RedirectScheme: config.RedirectScheme{
			Scheme:    "https",
			Permanent: true,
		},
	}
	mw.HTTP.Middlewares[config.HSTS] = &config.Middleware{
		Headers: config.Headers{
			STSSeconds: 31536000,
		},
	}

	if cfg.AuthorizationEndpoint != "" {
		mw.HTTP.Middlewares[config.FORWARDAUTH] = &config.Middleware{
			ForwardAuth: config.ForwardAuth{
				Address: fmt.Sprintf("http://localhost:%d/auth", port),
			},
		}
	}

	if err := mw.Write(path.Join(cfg.ConfigPath, "sys_middlewares.yaml")); err != nil {
		panic(err)
	}

	// Migrate certResolver for all configs to the specified one
	// if forward auth is disabled reflect this to all proxy entries
	names, err := config.ListNames(cfg.ConfigPath)
	check(err)
	for _, n := range names {
		path := path.Join(cfg.ConfigPath, n+".yaml")
		c, err := config.FromFile(path)
		check(err)
		if c.HTTP.Routers[n].TLS != nil {
			c.HTTP.Routers[n].TLS.CertResolver = cfg.ConfigOptions.CertResolver
			check(c.Write(path))
		}
		if cfg.AuthorizationEndpoint == "" {
			for _, r := range c.HTTP.Routers {
				r.Middlewares = splice(r.Middlewares, config.FORWARDAUTH+"@file")
			}
			check(c.Write(path))
		}
	}

	r := SetupRoutes(cfg)
	logger.Info(fmt.Sprintf("Starting server on :%d", port))
	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}

func splice(slice []string, needle string) []string {
	s := []string{}
	for _, i := range slice {
		if i != needle {
			s = append(s, i)
		}
	}
	return s
}
