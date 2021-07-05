package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pheelee/traefik-admin/config"
	"github.com/pheelee/traefik-admin/internal/server"
	"github.com/pheelee/traefik-admin/logger"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var port int
	var cfgpath string
	var certresolver string
	cfg := server.Config{}

	flag.StringVar(&cfgpath, "ConfigPath", "", "path where the dynamic config files getting stored")
	flag.StringVar(&cfg.WebRoot, "WebRoot", "", "defines the WebRoot containing index.html and static resources (for development)")
	flag.StringVar(&certresolver, "CertResolver", "http01", "name of the cert resolver which is configured for traefik, e.g http01 or dns01")
	flag.StringVar(&cfg.AuthorizationEndpoint, "AuthEndpoint", "", "indieauth authorization endpoint for auth forwarding, e.g https://homeassistant.tld/auth/authorize")
	flag.StringVar(&cfg.CookieSecret, "CookieSecret", "", "secret to encode session cookie (use strong random string)")
	flag.IntVar(&port, "Port", 8099, "Listening Port")

	flag.Parse()

	if cfgpath == "" || certresolver == "" || cfg.CookieSecret == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	config.Manager = config.ConfigManager{Path: cfgpath, CertResolver: certresolver}

	// Create sys configs
	mw := config.Config{
		Path: path.Join(cfgpath, "sys_middlewares.yaml"),
		HTTP: config.HTTP{
			Middlewares: make(map[string]*config.Middleware),
		},
	}

	mw.HTTP.Middlewares[strings.Replace(config.REDIRSCHEME, "@file", "", -1)] = &config.Middleware{
		RedirectScheme: config.RedirectScheme{
			Scheme:    "https",
			Permanent: true,
		},
	}
	mw.HTTP.Middlewares[strings.Replace(config.HSTS, "@file", "", -1)] = &config.Middleware{
		Headers: config.Headers{
			STSSeconds: 31536000,
		},
	}

	if cfg.AuthorizationEndpoint != "" {
		mw.HTTP.Middlewares[strings.Replace(config.FORWARDAUTH, "@file", "", -1)] = &config.Middleware{
			ForwardAuth: config.ForwardAuth{
				Address: fmt.Sprintf("http://localhost:%d/auth", port),
			},
		}
	}

	check(mw.Save())

	// Add unique id for all configs
	check(config.Manager.MigrateConfig())
	// Migrate certResolver for all configs to the specified one
	check(config.Manager.SetCertResolver(certresolver))
	// if forward auth is disabled reflect this to all proxy entries
	if cfg.AuthorizationEndpoint == "" {
		check(config.Manager.SetForwardAuth(config.Remove))
	}

	r := server.SetupRoutes(cfg)
	logger.Info(fmt.Sprintf("Starting server on :%d", port))
	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
