package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pheelee/traefik-admin/config"
	"github.com/pheelee/traefik-admin/logger"
)

var appcfg appConfig

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
	if cfg, err = config.Get(path.Join(appcfg.ConfigPath, name+".yaml")); err != nil {
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
		b       []byte
		err     error
		ret     bool
		cfgopts config.UserInput
		vErrs   config.ValidationError = config.ValidationError{Field: make(map[string]string), Generic: []string{}}
	)

	name := mux.Vars(r)["name"]
	w.Header().Set("content-type", "application/json")

	switch r.Method {
	case "POST":
		// check if config already exists
		if config.Exists(path.Join(appcfg.ConfigPath, name+".yaml")) {
			vErrs.Field["name"] = "Duplicate names not allowed"
			b, _ = json.Marshal(vErrs)
			w.WriteHeader(http.StatusConflict)
			w.Write(b)
			return
		}
	case "PUT":
		// check if config exists
		if !config.Exists(path.Join(appcfg.ConfigPath, name+".yaml")) {
			vErrs.Field["name"] = "Cannot rename config"
			b, _ = json.Marshal(vErrs)
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
		b, _ = json.Marshal(vErrs)
		logger.Error(err)
		w.Write(b)
		return
	}
	if err = json.Unmarshal(b, &cfgopts); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, _ = json.Marshal(vErrs)
		logger.Error(err)
		w.Write(b)
		return
	}

	// Validate user input

	if ret, vErrs = cfgopts.Validate(); !ret {
		w.WriteHeader(http.StatusBadRequest)
		b, _ = json.Marshal(vErrs)
		w.Write(b)
		return
	}

	cfg, err := config.Create(path.Join(appcfg.ConfigPath, name+".yaml"), name, cfgopts)
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

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		srcIP := r.RemoteAddr
		fwdIP := r.Header.Get("X-Forwarded-For")
		if len(fwdIP) > 0 {
			srcIP = strings.Split(fwdIP, " ")[0]
		}

		logger.Info(fmt.Sprintf("%s %s %s %dms", r.Method, r.URL.Path, srcIP, time.Now().Sub(t1).Milliseconds()))
	})
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

// SetupRoutes connects the functions to the endpoints
func SetupRoutes(cfg appConfig) http.Handler {
	appcfg = cfg
	mux := mux.NewRouter()
	fs := http.FileServer(http.Dir(cfg.WebRoot))
	mux.Use(recovery)
	mux.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	mux.HandleFunc("/config/", getAll).Methods("GET")
	mux.HandleFunc("/config/{name}", getConfig).Methods("GET")
	mux.HandleFunc("/config/{name}", saveConfig).Methods("POST", "PUT")
	mux.HandleFunc("/config/{name}", deleteConfig).Methods("DELETE")

	mux.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(appcfg.WebRoot, "index.html"))
	})

	loggedRouter := handlers.LoggingHandler(os.Stdout, mux)
	return loggedRouter
}
