package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/pheelee/traefik-admin/config"
	"github.com/pheelee/traefik-admin/internal/indieauth"
	"github.com/pheelee/traefik-admin/logger"
)

//go:embed webrootSrc
var efs embed.FS

var appcfg appConfig

var cookieStore *sessions.CookieStore

var assetHashes sync.Map

func getAll(w http.ResponseWriter, r *http.Request) {
	var (
		configList []config.UserInput
		err        error
		b          []byte
	)

	if configList, err = config.GetAllUserInput(appcfg.ConfigPath); err != nil {
		panic(err)
	}
	if b, err = json.Marshal(configList); err != nil {
		panic(err)
	}
	w.Header().Set("content-type", "application/json")
	w.Write(b)
}

func getConfig(w http.ResponseWriter, r *http.Request) {
	var (
		cfg *config.Config
		err error
		b   []byte
	)
	name := mux.Vars(r)["name"]
	if cfg, err = config.FromFile(path.Join(appcfg.ConfigPath, name+".yaml")); err != nil {
		panic(err)
	}
	if b, err = json.Marshal(cfg); err != nil {
		panic(err)
	}
	w.Header().Set("content-type", "application/json")
	w.Write(b)
}

func saveConfig(w http.ResponseWriter, r *http.Request) {

	var (
		b          []byte
		err        error
		cfgopts    config.UserInput
		validation config.Validation = config.NewValidation()
	)

	name := mux.Vars(r)["name"]
	w.Header().Set("content-type", "application/json")

	switch r.Method {
	case "POST":
		// check if config already exists
		if config.Exists(path.Join(appcfg.ConfigPath, name+".yaml")) {
			validation.Errors.Name = "Duplicate names not allowed"
			validation.Valid = false
			b, _ = json.Marshal(validation)
			w.WriteHeader(http.StatusConflict)
			w.Write(b)
			return
		}
	case "PUT":
		// check if config exists
		if !config.Exists(path.Join(appcfg.ConfigPath, name+".yaml")) {
			validation.Errors.Name = "Cannot rename config"
			validation.Valid = false
			b, _ = json.Marshal(validation)
			w.WriteHeader(http.StatusNotFound)
			w.Write(b)
			return
		}
	}

	// Parse User input
	b, err = ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		validation.Valid = false
		b, _ = json.Marshal(validation)
		logger.Error(err)
		w.Write(b)
		return
	}
	if err = json.Unmarshal(b, &cfgopts); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		validation.Valid = false
		b, _ = json.Marshal(validation)
		logger.Error(err)
		w.Write(b)
		return
	}

	// Validate user input

	if validation = cfgopts.Validate(); !validation.Valid {
		w.WriteHeader(http.StatusBadRequest)
		b, _ = json.Marshal(validation)
		w.Write(b)
		return
	}

	cfg, err := config.Create(path.Join(appcfg.ConfigPath, name+".yaml"), name, cfgopts, appcfg.ConfigOptions)
	if err != nil {
		panic(err)
	}
	cfgJSON := cfg.ToUserInput(name)
	b, _ = json.Marshal(cfgJSON)
	w.Write(b)
}

func deleteConfig(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := config.Delete(path.Join(appcfg.ConfigPath, name+".yaml")); err != nil {
		panic(err)
	}
}

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				logger.Error(err)
				jsonBody, _ := json.Marshal(map[string]string{
					"error": "There was an internal server error",
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requireAjax(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type features struct {
	ForwardAuth forwardauth `json:"forwardauth"`
}

type forwardauth struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

func getFeatures(w http.ResponseWriter, r *http.Request) {
	f := features{
		ForwardAuth: forwardauth{
			Enabled: appcfg.AuthorizationEndpoint != "",
			URL:     appcfg.AuthorizationEndpoint,
		},
	}
	b, _ := json.Marshal(f)
	w.Write(b)
}

func embedAsset(next http.Handler, uri string, embed string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.Replace(r.URL.Path, uri, embed, -1)
		fpath := strings.TrimPrefix(r.URL.Path, "/")
		b, err := efs.ReadFile(fpath)
		if err != nil {
			//TODO: maybe implement more fine grained errors like 404
			panic(err)
		}
		h, ok := assetHashes.Load(fpath)
		if !ok {
			m := sha256.New()
			m.Write(b)
			h = hex.EncodeToString(m.Sum(nil))
			assetHashes.Store(fpath, h)
		}
		inm := r.Header.Get("If-None-Match")
		if inm != "" && inm == h.(string) {
			w.Header().Set("ETag", h.(string))
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", h.(string))
		next.ServeHTTP(w, r)

	})
}

// SetupRoutes connects the functions to the endpoints
func SetupRoutes(cfg appConfig) http.Handler {
	var fs http.Handler
	appcfg = cfg
	mux := mux.NewRouter()
	mux.Use(recovery)

	// setup indieauth
	if appcfg.AuthorizationEndpoint != "" {
		cookieStore = sessions.NewCookieStore([]byte(appcfg.CookieSecret))
		ia, err := indieauth.New(cookieStore, "http://localhost/endpoints", appcfg.AuthorizationEndpoint)
		if err != nil {
			panic(err)
		}

		iaMiddleware := ia.Middleware()
		mux.HandleFunc(indieauth.DefaultRedirectPath, ia.RedirectHandler)
		mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
			ia.Logout(w, r)
		})

		mux.Handle("/auth/verify", iaMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("authorized"))
		})))

		mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
			uri := r.Header.Get("X-Forwarded-Uri")
			if strings.HasPrefix(uri, indieauth.DefaultRedirectPath) {
				r.URL, _ = url.Parse(uri)
			} else {
				r.URL.Path = "/auth/verify"
			}
			mux.ServeHTTP(w, r)
		})

		logger.Info(fmt.Sprintf("enabling forward-auth using endpoint %s", appcfg.AuthorizationEndpoint))
	} else {
		logger.Info("forward-auth middleware not enabled because no authorization endpoint was specified")
	}

	cfgmux := mux.PathPrefix("/config").Subrouter()
	cfgmux.Use(requireAjax)
	cfgmux.HandleFunc("/", getAll).Methods("GET")
	cfgmux.HandleFunc("/{name}", getConfig).Methods("GET")
	cfgmux.HandleFunc("/{name}", saveConfig).Methods("POST", "PUT")
	cfgmux.HandleFunc("/{name}", deleteConfig).Methods("DELETE")
	mux.HandleFunc("/features", getFeatures).Methods("GET")

	if cfg.WebRoot != "" {
		fs = http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.WebRoot)))
	} else {
		fs = embedAsset(http.FileServer(http.FS(efs)), "static", "webrootSrc")
	}
	mux.PathPrefix("/static/").Handler(fs)

	mux.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if appcfg.WebRoot != "" {
			http.ServeFile(w, r, path.Join(appcfg.WebRoot, "index.html"))
		} else {
			b, err := efs.ReadFile("webrootSrc/index.html")
			if err != nil {
				panic(err)
			}
			w.Write(b)
		}
	})

	loggedRouter := handlers.LoggingHandler(os.Stdout, mux)
	return loggedRouter
}
