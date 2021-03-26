package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/bcrypt"
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

// Options defines various static settings for config generation
type Options struct {
	CertResolver string
}

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

//ToUserInput converts a config to the struct used by the frontend
func (c *Config) ToUserInput(name string) UserInput {
	u := UserInput{
		Name:          name,
		Domain:        strings.TrimSuffix(strings.TrimPrefix(c.HTTP.Routers[name].Rule, "Host(`"), "`)"),
		Backend:       c.HTTP.Services[name].LoadBalancer.Servers[0].URL,
		ForwardAuth:   c.HTTP.containsRouter(name+"-http") && c.HTTP.Routers[name+"-http"].hasMiddleware(FORWARDAUTH+"@file"),
		HTTPS:         c.HTTP.Routers[name].TLS != nil,
		ForceTLS:      c.HTTP.containsRouter(name+"-http") && c.HTTP.Routers[name+"-http"].hasMiddleware(REDIRSCHEME+"@file"),
		HSTS:          c.HTTP.Routers[name].hasMiddleware(HSTS + "@file"),
		Headers:       make([]headersInput, 5),
		BasicAuth:     make([]basicAuthInput, 5),
		IPRestriction: &ipRestriction{Depth: 0, IPs: make([]string, 5)},
	}
	headers, ok := c.HTTP.Middlewares[name+"-headers"]
	if ok {
		i := 0
		for n, v := range headers.Headers.CustomRequestHeaders {
			u.Headers[i] = headersInput{Name: n, Value: v}
			i++
		}
	}
	auth, ok := c.HTTP.Middlewares[name+"-basicauth"]
	if ok {
		for i, entry := range auth.BasicAuth.Users {
			raw := strings.Split(entry, ":")
			u.BasicAuth[i] = basicAuthInput{Username: raw[0], Password: raw[1]}
		}
	}
	iprestriction, ok := c.HTTP.Middlewares[name+"-iprestrict"]
	if ok {
		if iprestriction.IPWhiteList.IPStrategy != nil {
			u.IPRestriction.Depth = iprestriction.IPWhiteList.IPStrategy.Depth
		}
		// fill remaining slots with empty strings
		for i := 0; i < len(u.IPRestriction.IPs); i++ {
			if len(iprestriction.IPWhiteList.SourceRange) > i {
				u.IPRestriction.IPs[i] = iprestriction.IPWhiteList.SourceRange[i]
			}
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

// FromFile returns the requested config
func FromFile(cfgPath string) (*Config, error) {
	var (
		b   []byte
		err error
		cfg Config
	)
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
		configFiles []string
		configList  []UserInput = []UserInput{}
		cfg         *Config
		err         error
		b           []byte
	)
	if configFiles, err = ListNames(cfgPath); err != nil {
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
		configList = append(configList, cfg.ToUserInput(c))
	}
	return configList, nil
}

// Create writes a new config
func Create(cfgPath string, name string, c UserInput, o Options) (*Config, error) {
	var (
		b   []byte
		err error
	)
	cfg := New(name, c)

	// add or remove config options based on user inputs
	switch c.HTTPS {
	case true:
		cfg.HTTP.Routers[name].TLS = &routerTLSConfig{CertResolver: o.CertResolver}
		cfg.HTTP.Routers[name].Entrypoints = []string{"websecure"}
		cfg.HTTP.Routers[name+"-http"] = &Router{
			Entrypoints: []string{"web"},
			Rule:        cfg.HTTP.Routers[name].Rule,
			Service:     cfg.HTTP.Routers[name].Service,
		}

		//add redirect middleware
		if c.ForceTLS {
			cfg.HTTP.Routers[name+"-http"].Middlewares = append(cfg.HTTP.Routers[name+"-http"].Middlewares, REDIRSCHEME+"@file")
		}
		// enable HSTS
		if c.HSTS {
			cfg.HTTP.Routers[name].Middlewares = append(cfg.HTTP.Routers[name].Middlewares, HSTS+"@file")
		}
	case false:
		cfg.HTTP.Routers[name].TLS = nil
		cfg.HTTP.Routers[name].Entrypoints = []string{"web"}
	}

	// add forward auth middleware
	if c.ForwardAuth {
		for _, r := range cfg.HTTP.Routers {
			r.Middlewares = append(r.Middlewares, FORWARDAUTH+"@file")
		}
	}

	// do we have some headers?
	headerMW := Headers{}
	headerMW.fromInput(c)
	if len(headerMW.CustomRequestHeaders) > 0 {
		cfg.HTTP.Middlewares[name+"-headers"] = &Middleware{Headers: headerMW}
		for _, r := range cfg.HTTP.Routers {
			r.Middlewares = append(r.Middlewares, name+"-headers")
		}
	}

	// do we have basic auth?
	var users []string = make([]string, 0)
	for _, ba := range c.BasicAuth {
		if ba.Username != "" {
			hash, _ := bcrypt.GenerateFromPassword([]byte(ba.Password), bcrypt.DefaultCost)
			users = append(users, ba.Username+":"+string(hash))
		}
	}
	if len(users) > 0 {
		cfg.HTTP.Middlewares[name+"-basicauth"] = &Middleware{BasicAuth: BasicAuth{}}
		cfg.HTTP.Middlewares[name+"-basicauth"].BasicAuth.Users = users
		cfg.HTTP.Routers[name].Middlewares = append(cfg.HTTP.Routers[name].Middlewares, name+"-basicauth")
	}

	// do we have any ip restrictions?
	ipr := getNonEmpty(c.IPRestriction.IPs)
	if c.IPRestriction != nil && len(ipr) > 0 {
		mw := &Middleware{IPWhiteList: IPWhiteList{SourceRange: ipr}}
		if c.IPRestriction.Depth > 0 {
			mw.IPWhiteList.IPStrategy = &IPStrategy{Depth: c.IPRestriction.Depth}
		}
		cfg.HTTP.Middlewares[name+"-iprestrict"] = mw
		for _, r := range cfg.HTTP.Routers {
			r.Middlewares = append(r.Middlewares, name+"-iprestrict")
		}
	}

	// Serialize and write the yaml config file
	if b, err = yaml.Marshal(cfg); err != nil {
		return nil, err
	}
	if err = ioutil.WriteFile(cfgPath, b, 0666); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ListNames returns all configs in a directory
func ListNames(cfgPath string) ([]string, error) {
	var (
		cfgs []string
		err  error
		fi   []os.FileInfo
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

func getNonEmpty(slice []string) []string {
	o := make([]string, 0)
	for _, s := range slice {
		if s != "" {
			o = append(o, s)
		}
	}
	return o
}
