package config

import (
	"fmt"

	"github.com/pheelee/traefik-admin/helpers"
	"github.com/pheelee/traefik-admin/logger"
)

const (
	FORWARDAUTH = "sys-forwardauth@file"
	REDIRSCHEME = "sys-redirscheme@file"
	HSTS        = "sys-hsts@file"
)

// Middleware defines Traefik Middleware
type Middleware struct {
	RedirectScheme RedirectScheme `yaml:"redirectScheme,omitempty"`
	Headers        Headers        `yaml:"headers,omitempty"`
	BasicAuth      BasicAuth      `yaml:"basicAuth,omitempty"`
	IPWhiteList    IPWhiteList    `yaml:"ipWhiteList,omitempty"`
	ForwardAuth    ForwardAuth    `yaml:"forwardAuth,omitempty"`
}

// RedirectScheme holds data for a schema redirect
type RedirectScheme struct {
	Scheme    string `yaml:"scheme"`
	Permanent bool   `yaml:"permanent"`
}

// Headers hold custom headers structure
type Headers struct {
	CustomRequestHeaders map[string]string `yaml:"customRequestHeaders,omitempty"`
	STSSeconds           int64             `yaml:"stsSeconds,omitempty"`
}

//BasicAuth holds data for basic authentication
type BasicAuth struct {
	Users        []string `yaml:"users,omitempty"`
	Realm        string   `yaml:"realm,omitempty"`
	HeaderField  string   `yaml:"headerField,omitempty"`
	RemoveHeader bool     `yaml:"removeHeader,omitempty"`
}

// IPWhiteList holds the ip white list configuration.
type IPWhiteList struct {
	SourceRange []string    `yaml:"sourceRange,omitempty"`
	IPStrategy  *IPStrategy `yaml:"ipStrategy,omitempty"`
}

// ForwardAuth holds the forward auth data
type ForwardAuth struct {
	Address string `yaml:"address,omitempty"`
}

// IPStrategy holds the ip strategy configuration.
type IPStrategy struct {
	Depth int `yaml:"depth,omitempty"`
}

func (h *Headers) fromInput(c *UserInput) {
	h.CustomRequestHeaders = make(map[string]string)
	for _, uh := range c.Headers {
		if uh.Name != "" {
			switch uh.Value {
			case "$ServerIP":
				ip, err := helpers.GetHostIP()
				if err != nil {
					logger.Warning(fmt.Sprintf("could not get ip address requested for header %s", uh.Name))
					h.CustomRequestHeaders[uh.Name] = "n/a"
				} else {
					h.CustomRequestHeaders[uh.Name] = ip
				}

			default:
				h.CustomRequestHeaders[uh.Name] = uh.Value
			}
		}
	}
}
