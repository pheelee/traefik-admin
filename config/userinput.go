package config

import "regexp"

// UserInput hold the data submitted by the api request
type UserInput struct {
	Name     string         `json:"name"`
	Domain   string         `json:"domain"`
	Backend  string         `json:"backend"`
	HTTPS    bool           `json:"https"`
	ForceTLS bool           `json:"forcetls"`
	Headers  []headersInput `json:"headers"`
}

type headersInput struct {
	Name  string
	Value string
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

	if len(u.Headers) > 20 {
		pass = false
		errs.Generic = append(errs.Generic, "Too many headers, max 20 allowed")
	}

	return pass, errs
}
