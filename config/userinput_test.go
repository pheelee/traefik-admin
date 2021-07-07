package config

import "testing"

func TestConnect(t *testing.T) {
	// Perfom panic testing
	b := Backend{URL: "http://127.0.0.1:8090"}
	b.Connect()
	b = Backend{URL: "127.0.0.1:8090"}
	b.Connect()
	b = Backend{URL: "tcp://127.0.0.1:8090"}
	b.Connect()
	b = Backend{URL: "265.123.562.123:8090"}
	b.Connect()
	b = Backend{URL: "https://123124124124:8090"}
	b.Connect()
}

func TestValidation(t *testing.T) {
	ui.Name = "123.Test"
	ui.Domain = "1241234"
	ui.Backend = Backend{URL: "test.example.com"}
	ui.BasicAuth[0].Username = "aerg.1241t$"
	ui.BasicAuth[0].Password = ""
	ui.BasicAuth = append(ui.BasicAuth, basicAuthInput{})
	ui.IPRestriction = &ipRestriction{
		Depth: 32,
		IPs: []string{
			"1234.654.14.45",
			"123.154.14.45/78",
		},
	}
	ui.Headers = append(ui.Headers, headersInput{Name: "aewrg@awerg", Value: "FO"})
	ui.Headers = append(ui.Headers, headersInput{})
	v := ui.Validate()
	if v.Valid {
		t.Error("Should be invalid")
	}
}
