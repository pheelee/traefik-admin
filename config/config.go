package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config holds a dynamic traefik config
type Config struct {
	HTTP http `yaml:"http"`
}

type http struct {
	Routers  map[string]*Router  `yaml:"routers,omitempty"`
	Services map[string]*Service `yaml:"services,omitempty"`
}

// Router holds the config part for the router
type Router struct {
	Entrypoints []string         `yaml:"entryPoints,omitempty"`
	Rule        string           `yaml:"rule"`
	Service     string           `yaml:"service"`
	TLS         *routerTLSConfig `yaml:"tls"`
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

// UserInput hold the data submitted by the api request
type UserInput struct {
	Name    string `json:"name"`
	Domain  string `json:"domain"`
	Backend string `json:"backend"`
}

//ValidationError provides information about invalid fields
type ValidationError struct {
	Field map[string]string
}

var (
	fi          []os.FileInfo
	err         error
	b           []byte
	cfg         Config
	configFiles []string
)

func (c *Config) toUserInput(name string) UserInput {
	return UserInput{
		Name:    name,
		Domain:  strings.TrimSuffix(strings.TrimPrefix(c.HTTP.Routers[name].Rule, "Host(`"), "`)"),
		Backend: c.HTTP.Services[name].LoadBalancer.Servers[0].URL,
	}
}

// Validate checks userinput against rules
func (u *UserInput) Validate() (bool, ValidationError) {
	var (
		match bool
		pass  bool = true
		errs  ValidationError
	)
	errs = ValidationError{Field: make(map[string]string)}
	if match, _ = regexp.MatchString("^[a-zA-Z0-9]{3,32}$", u.Name); !match {
		pass = false
		errs.Field["name"] = "String between 3 and 32 chars required" //append(errs, ValidationError{Field: "name", Message: "Name has invalid format"})
	}

	if match, _ = regexp.MatchString("^([a-zA-Z0-9]+\\.){2,63}[a-zA-Z]{2,6}$", u.Domain); !match {
		pass = false
		errs.Field["domain"] = "not a valid domain name" //append(errs, ValidationError{Field: "domain", Message: "Domain has invalid format"})
	}

	if match, _ = regexp.MatchString("^http(s)?:\\/\\/[a-zA-Z0-9.]+:\\d{0,5}$", u.Backend); !match {
		pass = false
		errs.Field["backend"] = "Format: http://192.168.1.12:5000" // append(errs, ValidationError{Field: "backend", Message: "Backend has invalid format"})
	}

	return pass, errs
}

// New returns an initialized new config
func New(service string, c UserInput) Config {
	return Config{
		HTTP: http{
			Routers: map[string]*Router{
				service: {
					Entrypoints: []string{"websecure"},
					Service:     service,
					TLS:         &routerTLSConfig{CertResolver: "http01"},
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
		cfgs = append(cfgs, strings.TrimSuffix(f.Name(), path.Ext(f.Name())))
	}
	return cfgs, nil
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
