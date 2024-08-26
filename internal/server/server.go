package server

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

var VERSION string

//go:embed webrootSrc
var efs embed.FS

var appcfg Config

var cookieStore *sessions.CookieStore

var assetHashes sync.Map

type Config struct {
	WebRoot               string
	AuthorizationEndpoint string
	CookieSecret          string
}

func List(w http.ResponseWriter, r *http.Request) {
	var (
		configList []config.UserInput
		err        error
		b          []byte
	)

	if configList, err = config.Manager.ListUserInputs(); err != nil {
		panic(err)
	}

	// perform health checks
	var wg sync.WaitGroup
	for i, _ := range configList {
		wg.Add(1)
		go func(wg *sync.WaitGroup, b *config.Backend) {
			defer wg.Done()
			b.Connect()
		}(&wg, &configList[i].Backend)
	}
	wg.Wait()

	if b, err = json.Marshal(configList); err != nil {
		panic(err)
	}
	w.Header().Set("content-type", "application/json")
	w.Write(b)
}

func Get(w http.ResponseWriter, r *http.Request) {
	var (
		cfg *config.Config
		err error
		b   []byte
	)
	id := mux.Vars(r)["id"]
	cfg = config.Manager.Get(id)
	if cfg == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if b, err = json.Marshal(cfg); err != nil {
		panic(err)
	}
	w.Header().Set("content-type", "application/json")
	w.Write(b)
}

func Save(w http.ResponseWriter, r *http.Request) {

	var (
		b   []byte
		err error
		u   *config.UserInput
		c   *config.Config
		v   config.Validation = config.NewValidation()
	)
	w.Header().Set("content-type", "application/json")

	// Parse User input
	b, err = ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		v.Valid = false
		b, _ = json.Marshal(v)
		logger.Error(err)
		w.Write(b)
		return
	}
	if err = json.Unmarshal(b, &u); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		v.Valid = false
		b, _ = json.Marshal(v)
		logger.Error(err)
		w.Write(b)
		return
	}

	// Validate user input
	if v = u.Validate(); !v.Valid {
		w.WriteHeader(http.StatusBadRequest)
		b, _ = json.Marshal(v)
		w.Write(b)
		return
	}

	switch r.Method {
	case "POST":
		c, err = config.Manager.Add(u)
	case "PUT":
		c, err = config.Manager.Update(u)
	}

	if err != nil {
		panic(err)
	}
	u, err = c.ToUserInput()
	if err != nil {
		panic(err)
	}
	u.Backend.Connect()
	b, _ = json.Marshal(u)
	w.Write(b)
}

func Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := config.Manager.Delete(id); err != nil {
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
	Version     string      `json:"version"`
}

type forwardauth struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

func Features(w http.ResponseWriter, r *http.Request) {
	f := features{
		ForwardAuth: forwardauth{
			Enabled: appcfg.AuthorizationEndpoint != "",
			URL:     appcfg.AuthorizationEndpoint,
		},
	}
	if VERSION == "" {
		VERSION = "dev"
	}
	f.Version = VERSION
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

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "sameorigin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "script-src 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "script-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// SetupRoutes connects the functions to the endpoints
func SetupRoutes(cfg Config) http.Handler {
	var fs http.Handler
	appcfg = cfg
	mux := mux.NewRouter()
	mux.Use(recovery, securityHeaders)

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
	cfgmux.HandleFunc("/", List).Methods("GET")
	cfgmux.HandleFunc("/{id}", Get).Methods("GET")
	cfgmux.HandleFunc("/{id}", Save).Methods("POST", "PUT")
	cfgmux.HandleFunc("/{id}", Delete).Methods("DELETE")
	mux.HandleFunc("/features", Features).Methods("GET")

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
