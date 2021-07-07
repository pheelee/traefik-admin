package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

// Config holds a dynamic traefik config
type Config struct {
	Path   string `yaml:"-"`
	id     string `yaml:"-"`
	loaded bool   `yaml:"-"`
	HTTP   HTTP   `yaml:"http"`
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

func (c *Config) Load() error {
	if !c.loaded {
		b, err := ioutil.ReadFile(c.Path)
		if err != nil {
			return err
		}
		if err = yaml.Unmarshal(b, c); err != nil {
			return err
		}
		c.loaded = true
		return nil
	}
	return nil
}

//Name is the first part of the filename name_hash.yaml
func (c *Config) Name() string {
	p := strings.Split(c.Path, "/")
	name := p[len(p)-1]
	name = strings.Replace(name, path.Ext(name), "", -1)
	k := strings.Split(name, "_")
	return k[0]
}

func (c *Config) ID() string {
	p := strings.Split(c.Path, "/")
	name := p[len(p)-1]
	return strings.Replace(name, path.Ext(name), "", -1)
}

func (h *HTTP) containsRouter(name string) bool {
	_, ok := h.Routers[name]
	return ok
}

func (h *HTTP) hasAnyRouterMiddleware(name string) bool {
	for _, r := range h.Routers {
		if r.hasMiddleware(name) {
			return true
		}
	}
	return false
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
func (c *Config) ToUserInput() (*UserInput, error) {
	if err := c.Load(); err != nil {
		return nil, err
	}
	id := c.ID()
	u := &UserInput{
		ID:            id,
		Name:          c.Name(),
		Domain:        strings.TrimSuffix(strings.TrimPrefix(c.HTTP.Routers[id+"-http"].Rule, "Host(`"), "`)"),
		Backend:       Backend{URL: c.HTTP.Services[id].LoadBalancer.Servers[0].URL},
		ForwardAuth:   c.HTTP.hasAnyRouterMiddleware(FORWARDAUTH),
		HTTPS:         c.HTTP.containsRouter(id) && c.HTTP.Routers[id].TLS != nil,
		ForceTLS:      c.HTTP.containsRouter(id+"-http") && c.HTTP.Routers[id+"-http"].hasMiddleware(REDIRSCHEME),
		HSTS:          c.HTTP.containsRouter(id) && c.HTTP.Routers[id].hasMiddleware(HSTS),
		Headers:       make([]headersInput, 5),
		BasicAuth:     make([]basicAuthInput, 5),
		IPRestriction: &ipRestriction{Depth: 0, IPs: make([]string, 5)},
	}
	headers, ok := c.HTTP.Middlewares[id+"-headers"]
	if ok {
		i := 0
		for n, v := range headers.Headers.CustomRequestHeaders {
			u.Headers[i] = headersInput{Name: n, Value: v}
			i++
		}
	}
	auth, ok := c.HTTP.Middlewares[id+"-basicauth"]
	if ok {
		for i, entry := range auth.BasicAuth.Users {
			raw := strings.Split(entry, ":")
			u.BasicAuth[i] = basicAuthInput{Username: raw[0], Password: raw[1]}
		}
	}
	iprestriction, ok := c.HTTP.Middlewares[id+"-iprestrict"]
	if ok {
		if iprestriction.IPWhiteList.IPStrategy != nil {
			u.IPRestriction.Depth = iprestriction.IPWhiteList.IPStrategy.Depth
		}
		for i := 0; i < len(u.IPRestriction.IPs); i++ {
			if len(iprestriction.IPWhiteList.SourceRange) > i {
				u.IPRestriction.IPs[i] = iprestriction.IPWhiteList.SourceRange[i]
			}
		}
	}
	return u, nil
}

func FromUserInput(u *UserInput, certresolver string) *Config {
	if !u.Validate().Valid {
		return nil
	}
	c := &Config{
		id: u.Name + "_" + RandHash(),
		HTTP: HTTP{
			Routers:     map[string]*Router{},
			Services:    make(map[string]*Service),
			Middlewares: make(map[string]*Middleware),
		},
	}
	// Always add service
	c.HTTP.Services[c.id] = &Service{
		LoadBalancer: loadbalancer{
			Servers: []server{
				{
					URL: u.Backend.URL,
				},
			},
		},
	}
	// Always add http router
	c.HTTP.Routers[c.id+"-http"] = &Router{
		Entrypoints: []string{"web"},
		Service:     c.id,
		Rule:        fmt.Sprintf("Host(`%s`)", u.Domain),
		Middlewares: []string{},
	}
	// https redirect middleware if specified
	if u.ForceTLS {
		c.HTTP.Routers[c.id+"-http"].Middlewares = append(c.HTTP.Routers[c.id+"-http"].Middlewares, REDIRSCHEME)
	}
	// https router if enabled
	if u.HTTPS {
		c.HTTP.Routers[c.id] = &Router{
			Entrypoints: []string{"websecure"},
			Rule:        fmt.Sprintf("Host(`%s`)", u.Domain),
			Service:     c.id,
			TLS: &routerTLSConfig{
				CertResolver: certresolver,
			},
			Middlewares: []string{},
		}

		if u.HSTS {
			c.HTTP.Routers[c.id].Middlewares = append(c.HTTP.Routers[c.id].Middlewares, HSTS)
		}
	}

	// now we have stuff for both routers
	// add forward auth middleware
	if u.ForwardAuth {
		for _, r := range c.HTTP.Routers {
			r.Middlewares = append(r.Middlewares, FORWARDAUTH)
		}
	}

	// do we have some headers?
	headerMW := Headers{}
	headerMW.fromInput(u)
	if len(headerMW.CustomRequestHeaders) > 0 {
		c.HTTP.Middlewares[c.id+"-headers"] = &Middleware{Headers: headerMW}
		for _, r := range c.HTTP.Routers {
			r.Middlewares = append(r.Middlewares, c.id+"-headers")
		}
	}

	// do we have basic auth?
	var users []string = make([]string, 0)
	for _, ba := range u.BasicAuth {
		if ba.Username != "" {
			hash, _ := bcrypt.GenerateFromPassword([]byte(ba.Password), bcrypt.DefaultCost)
			users = append(users, ba.Username+":"+string(hash))
		}
	}
	if len(users) > 0 {
		c.HTTP.Middlewares[c.id+"-basicauth"] = &Middleware{BasicAuth: BasicAuth{}}
		c.HTTP.Middlewares[c.id+"-basicauth"].BasicAuth.Users = users
		c.HTTP.Routers[c.id].Middlewares = append(c.HTTP.Routers[c.id].Middlewares, c.id+"-basicauth")
	}

	// do we have any ip restrictions?
	if u.IPRestriction != nil {
		ipr := spliceEmpty(u.IPRestriction.IPs)
		if len(ipr) > 0 {
			mw := &Middleware{IPWhiteList: IPWhiteList{SourceRange: ipr}}
			if u.IPRestriction.Depth > 0 {
				mw.IPWhiteList.IPStrategy = &IPStrategy{Depth: u.IPRestriction.Depth}
			}
			c.HTTP.Middlewares[c.id+"-iprestrict"] = mw
			for _, r := range c.HTTP.Routers {
				r.Middlewares = append(r.Middlewares, c.id+"-iprestrict")
			}
		}
	}
	return c
}

func (c *Config) ChangeIdentifier(old string, new string) {
	c.Load()
	rKeys := c.RouterKeys()
	for _, k := range rKeys {
		c.HTTP.Routers[k].Service = new
		if strings.HasPrefix(k, old) {
			nn := strings.Replace(k, old, new, -1)
			c.HTTP.Routers[nn] = c.HTTP.Routers[k]
			delete(c.HTTP.Routers, k)
		}
	}
	c.HTTP.Services[new] = c.HTTP.Services[old]
	delete(c.HTTP.Services, old)
	c.Save()
}

func (c *Config) RouterKeys() []string {
	var keys []string = make([]string, 0)
	for k := range c.HTTP.Routers {
		keys = append(keys, k)
	}
	return keys
}

// Save serializes the config to file
func (c *Config) Save() error {
	var (
		err error
		b   []byte
	)
	if b, err = yaml.Marshal(c); err != nil {
		return err
	}
	err = ioutil.WriteFile(c.Path, b, 0644)
	return err
}

func spliceEmpty(slice []string) []string {
	o := make([]string, 0)
	for _, s := range slice {
		if s != "" {
			o = append(o, s)
		}
	}
	return o
}

func RandHash() string {
	var b []byte = make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(sha256.New().Sum(b))[:8]
}
