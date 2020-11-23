package config

import (
	"fmt"

	"github.com/pheelee/traefik-admin/helpers"
	"github.com/pheelee/traefik-admin/logger"
)

// Middleware defines Traefik Middleware
type Middleware struct {
	RedirectScheme RedirectScheme `yaml:"redirectScheme,omitempty"`
	Headers        Headers        `yaml:"headers,omitempty"`
}

// RedirectScheme holds data for a schema redirect
type RedirectScheme struct {
	Scheme    string `yaml:"scheme"`
	Permanent bool   `yaml:"permanent"`
}

// Headers hold custom headers structure
type Headers struct {
	CustomRequestHeaders map[string]string `yaml:"customRequestHeaders,omitempty"`
}

func (h *Headers) fromInput(c UserInput) {
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

			case "":
				h.CustomRequestHeaders[uh.Name] = uh.Value
			}
		}
	}
}
