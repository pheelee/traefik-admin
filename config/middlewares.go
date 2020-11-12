package config

// Middleware defines Traefik Middleware
type Middleware struct {
	RedirectScheme RedirectScheme `yaml:"redirectScheme"`
}

// RedirectScheme holds data for a schema redirect
type RedirectScheme struct {
	Scheme    string `yaml:"scheme"`
	Permanent bool   `yaml:"permanent"`
}
