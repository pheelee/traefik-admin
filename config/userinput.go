package config

import (
	"net"
	"regexp"
	"strings"
	"time"
)

// UserInput hold the data submitted by the api request
type UserInput struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Domain        string           `json:"domain"`
	Backend       Backend          `json:"backend"`
	ForwardAuth   bool             `json:"forwardauth"`
	HTTPS         bool             `json:"https"`
	ForceTLS      bool             `json:"forcetls"`
	HSTS          bool             `json:"hsts"`
	Headers       []headersInput   `json:"headers"`
	BasicAuth     []basicAuthInput `json:"basicauth"`
	IPRestriction *ipRestriction   `json:"ipRestriction"`
}

type headersInput struct {
	Name  string
	Value string
}

type basicAuthInput struct {
	Username string
	Password string
}

type ipRestriction struct {
	Depth int      `json:"depth"`
	IPs   []string `json:"ips"`
}

type Validation struct {
	Valid  bool            `json:"valid"`
	Errors ValidationError `json:"errors"`
}

//ValidationError provides information about invalid fields
type ValidationError struct {
	Name      string      `json:"name"`
	Domain    string      `json:"domain"`
	Backend   string      `json:"backend"`
	BasicAuth []basicAuth `json:"basicauth"`
	AllowedIP allowedIP   `json:"allowedip"`
	Headers   []header    `json:"headers"`
}

type basicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type allowedIP struct {
	NoProxies string   `json:"noproxies"`
	IP        []string `json:"ip"`
}

type header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Backend struct {
	URL     string `json:"url"`
	Healthy bool   `json:"healthy"`
}

func (b *Backend) Connect() {
	addr := strings.Replace(b.URL, "https://", "", -1)
	addr = strings.Replace(addr, "http://", "", -1)
	c, err := net.DialTimeout("tcp", addr, 1*time.Second)
	b.Healthy = err == nil
	if c != nil {
		c.Close()
	}
}

func NewValidation() Validation {
	return Validation{
		Valid: true,
		Errors: ValidationError{
			Name: "", Domain: "", Backend: "",
			BasicAuth: make([]basicAuth, 5),
			AllowedIP: allowedIP{NoProxies: "", IP: make([]string, 5)},
			Headers:   make([]header, 5),
		},
	}
}

// Validate checks userinput against rules
func (u *UserInput) Validate() Validation {
	var (
		rex   *regexp.Regexp
		rex2  *regexp.Regexp
		match bool       = true
		v     Validation = NewValidation()
	)
	if match, _ = regexp.MatchString("^[a-zA-Z0-9-]{3,32}$", u.Name); !match {
		v.Valid = false
		v.Errors.Name = "String between 3 and 32 chars required"
	}

	if match, _ = regexp.MatchString("^([a-zA-Z0-9]+\\.){2,63}[a-zA-Z]{2,6}$", u.Domain); !match {
		v.Valid = false
		v.Errors.Domain = "not a valid domain name"
	}

	// ToDo: improve validation (regarding ip addresses)
	if match, _ = regexp.MatchString("^http(s)?:\\/\\/[a-zA-Z0-9.]+:\\d{0,5}$", u.Backend.URL); !match {
		v.Valid = false
		v.Errors.Backend = "Format: http://192.168.1.12:5000"
	}

	rex = regexp.MustCompile("^[a-zA-Z0-9]{3,32}$")
	rex2 = regexp.MustCompile("^.{1,128}$")
	for i, b := range u.BasicAuth {
		if b.Username == "" && b.Password == "" {
			continue
		}
		if match = rex.MatchString(b.Username); !match {
			v.Valid = false
			v.Errors.BasicAuth[i].Username = "Invalid username"
		}
		if match = rex2.MatchString(b.Password); !match {
			v.Valid = false
			v.Errors.BasicAuth[i].Password = "Invalid password or username missing"
		}
	}

	// IP Restriction checks
	rex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}(\/\d{2})?$`)
	if u.IPRestriction != nil {
		if !inBetween(u.IPRestriction.Depth, 0, 30) {
			v.Valid = false
			v.Errors.AllowedIP.NoProxies = "must be between 0 and 30"
		}
		for i, k := range u.IPRestriction.IPs {
			if match = rex.MatchString(k); !match && k != "" {
				v.Valid = false
				v.Errors.AllowedIP.IP[i] = "Invalid IP/Net"
			}
		}
	}

	rex = regexp.MustCompile("^[a-zA-Z0-9-_]{1,64}$")
	rex2 = regexp.MustCompile("^.{3,128}$")
	for i, h := range u.Headers {
		if h.Name == "" && h.Value == "" {
			continue
		}
		if match = rex.MatchString(h.Name); !match {
			v.Valid = false
			v.Errors.Headers[i].Name = "Invalid header name"
		}
		if match = rex2.MatchString(h.Value); !match {
			v.Valid = false
			v.Errors.Headers[i].Value = "Invalid header value or header name missing"
		}
	}

	return v
}

func inBetween(i, min, max int) bool {
	return i >= min && i <= max
}
