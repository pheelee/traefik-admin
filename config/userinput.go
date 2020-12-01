package config

import "regexp"

// UserInput hold the data submitted by the api request
type UserInput struct {
	Name          string           `json:"name"`
	Domain        string           `json:"domain"`
	Backend       string           `json:"backend"`
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

//ValidationError provides information about invalid fields
type ValidationError struct {
	Field   map[string]string
	Generic []string `json:"generic"`
}

// Validate checks userinput against rules
func (u *UserInput) Validate() (bool, ValidationError) {
	var (
		match bool
		pass  bool            = true
		errs  ValidationError = ValidationError{Field: make(map[string]string), Generic: []string{}}
	)
	if match, _ = regexp.MatchString("^[a-zA-Z0-9]{3,32}$", u.Name); !match {
		pass = false
		errs.Field["name"] = "String between 3 and 32 chars required"
	}

	if match, _ = regexp.MatchString("^([a-zA-Z0-9]+\\.){2,63}[a-zA-Z]{2,6}$", u.Domain); !match {
		pass = false
		errs.Field["domain"] = "not a valid domain name"
	}

	if match, _ = regexp.MatchString("^http(s)?:\\/\\/[a-zA-Z0-9.]+:\\d{0,5}$", u.Backend); !match {
		pass = false
		errs.Field["backend"] = "Format: http://192.168.1.12:5000"
	}

	for _, h := range u.Headers {
		if match, _ = regexp.MatchString("^[a-zA-Z0-9-_]{1,64}$", h.Name); !match {
			pass = false
			errs.Generic = append(errs.Generic, "Invalid header specified")
			break
		}
		if match, _ = regexp.MatchString("^.{3,128}$", h.Value); !match {
			pass = false
			errs.Generic = append(errs.Generic, "Invalid header specified")
			break
		}
	}

	for _, b := range u.BasicAuth {
		if match, _ = regexp.MatchString("^[a-zA-Z0-9]{3,32}$", b.Username); !match {
			pass = false
			errs.Generic = append(errs.Generic, "Basic auth contains invalid entries")
			break
		}
		if match, _ = regexp.MatchString("^.{1,128}$", b.Password); !match {
			pass = false
			errs.Generic = append(errs.Generic, "Basic auth contains invalid entries")
			break
		}
	}

	// IP Restriction checks
	if u.IPRestriction != nil {
		if !inBetween(u.IPRestriction.Depth, 0, 30) {
			pass = false
			errs.Generic = append(errs.Generic, "ipRestriction depth must be between 1 and 30")
		}
		for _, i := range u.IPRestriction.IPs {
			if match, _ = regexp.MatchString(`^(\d{1,3}\.){3}\d{1,3}(\/\d{2})?$`, i); !match {
				pass = false
				errs.Generic = append(errs.Generic, "ipRestriction contains invalid entries")
			}
		}
	}

	return pass, errs
}

func inBetween(i, min, max int) bool {
	return i >= min && i <= max
}
