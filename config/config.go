package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config holds a dynamic traefik config
type Config struct {
	HTTP HTTP `yaml:"http"`
}

// HTTP defines the http entry struct of traefik
type HTTP struct {
	Routers     map[string]*Router     `yaml:"routers,omitempty"`
	Services    map[string]*Service    `yaml:"services,omitempty"`
	Middlewares map[string]*Middleware `yaml:"middlewares,omitempty"`
}

// Router holds the config part for the router
type Router struct {
	Entrypoints []string         `yaml:"entryPoints,omitempty"`
	Rule        string           `yaml:"rule"`
	Service     string           `yaml:"service,omitempty"`
	TLS         *routerTLSConfig `yaml:"tls,omitempty"`
	Middlewares []string         `yaml:"middlewares,omitempty"`
}

type routerTLSConfig struct {
	CertResolver string `yaml:"certResolver"`
}

// Service holds the config part for service
type Service struct {
	LoadBalancer loadbalancer `yaml:"loadBalancer"`
}

type loadbalancer struct {
	Servers []server
}

type server struct {
	URL string `yaml:"url"`
}

var (
	fi          []os.FileInfo
	err         error
	b           []byte
	cfg         Config
	configFiles []string
)

func (h *HTTP) containsRouter(name string) bool {
	_, ok := h.Routers[name]
	return ok
}

func (r *Router) hasMiddleware(name string) bool {
	for _, m := range r.Middlewares {
		if m == name {
			return true
		}
	}
	return false
}

func (c *Config) toUserInput(name string) UserInput {
	u := UserInput{
		Name:     name,
		Domain:   strings.TrimSuffix(strings.TrimPrefix(c.HTTP.Routers[name].Rule, "Host(`"), "`)"),
		Backend:  c.HTTP.Services[name].LoadBalancer.Servers[0].URL,
		HTTPS:    c.HTTP.Routers[name].TLS != nil,
		ForceTLS: c.HTTP.containsRouter(name+"-http") && c.HTTP.Routers[name+"-http"].hasMiddleware("sys-redirscheme@file"),
		Headers:  []headersInput{},
	}
	headers, ok := c.HTTP.Middlewares[name+"-headers"]
	if ok {
		for n, v := range headers.Headers.CustomRequestHeaders {
			u.Headers = append(u.Headers, headersInput{Name: n, Value: v})
		}
	}
	return u
}

// New returns an initialized new config
func New(service string, c UserInput) Config {
	return Config{
		HTTP: HTTP{
			Routers: map[string]*Router{
				service: {
					Entrypoints: []string{"web"},
					Service:     service,
					TLS:         nil,
					Rule:        fmt.Sprintf("Host(`%s`)", c.Domain),
				},
			},
			Services: map[string]*Service{
				service: {
					LoadBalancer: loadbalancer{
						Servers: []server{
							{URL: c.Backend},
						},
					},
				},
			},
			Middlewares: make(map[string]*Middleware),
		},
	}
}

// Get returns the requested config
func Get(cfgPath string) (*Config, error) {
	b, err = ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetAllUserInput returns config options for all configs
func GetAllUserInput(cfgPath string) ([]UserInput, error) {
	var (
		configList []UserInput = []UserInput{}
		cfg        *Config
	)
	if configFiles, err = List(cfgPath); err != nil {
		return nil, err
	}
	for _, c := range configFiles {
		if b, err = ioutil.ReadFile(path.Join(cfgPath, c+".yaml")); err != nil {
			return nil, fmt.Errorf("config %s: %s", c, err.Error())
		}
		if err = yaml.Unmarshal(b, &cfg); err != nil {
			return nil, fmt.Errorf("config %s: %s", c, err.Error())
		}
		if cfg == nil {
			return nil, fmt.Errorf("invalid config %s", c)
		}
		configList = append(configList, cfg.toUserInput(c))
	}
	return configList, nil
}

// Create writes a new config
func Create(cfgPath string, name string, c UserInput) error {
	cfg := New(name, c)

	// add or remove config options based on user inputs
	switch c.HTTPS {
	case true:
		cfg.HTTP.Routers[name].TLS = &routerTLSConfig{CertResolver: "http01"}
		cfg.HTTP.Routers[name].Entrypoints = []string{"websecure"}
		cfg.HTTP.Routers[name+"-http"] = &Router{
			Entrypoints: []string{"web"},
			Rule:        cfg.HTTP.Routers[name].Rule,
			Service:     cfg.HTTP.Routers[name].Service,
		}

		//add redirect middleware
		if c.ForceTLS {
			cfg.HTTP.Routers[name+"-http"].Middlewares = append(cfg.HTTP.Routers[name+"-http"].Middlewares, "sys-redirscheme@file")
		}
	case false:
		cfg.HTTP.Routers[name].TLS = nil
		cfg.HTTP.Routers[name].Entrypoints = []string{"web"}
	}

	// do we need some headers?
	headerMW := Headers{}
	headerMW.fromInput(c)
	if len(headerMW.CustomRequestHeaders) > 0 {
		cfg.HTTP.Middlewares[name+"-headers"] = &Middleware{Headers: headerMW}
		for _, r := range cfg.HTTP.Routers {
			r.Middlewares = append(r.Middlewares, name+"-headers")
		}
	}

	// Serialize and write the yaml config file
	if b, err = yaml.Marshal(cfg); err != nil {
		return err
	}
	if err = ioutil.WriteFile(cfgPath, b, 0666); err != nil {
		return err
	}
	return nil
}

// List returns all configs in a directory
func List(cfgPath string) ([]string, error) {
	var (
		cfgs []string
	)
	cfgs = make([]string, 0)

	if fi, err = ioutil.ReadDir(cfgPath); err != nil {
		return nil, err
	}

	for _, f := range fi {
		if !strings.HasPrefix(f.Name(), "sys_") {
			cfgs = append(cfgs, strings.TrimSuffix(f.Name(), path.Ext(f.Name())))
		}
	}
	return cfgs, nil
}

// Write serializes the config to file
func (c *Config) Write(path string) error {
	var (
		err error
		b   []byte
	)
	if b, err = yaml.Marshal(c); err != nil {
		return err
	}
	err = ioutil.WriteFile(path, b, 0644)
	return err
}

// Delete removes a config from the directory
func Delete(cfgPath string) error {
	return os.Remove(cfgPath)
}

// Exists checks the existence of a config
func Exists(cfgPath string) bool {
	_, err := os.Stat(cfgPath)
	return err == nil
}
