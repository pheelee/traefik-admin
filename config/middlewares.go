package config

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
			h.CustomRequestHeaders[uh.Name] = uh.Value
		}
	}
}
