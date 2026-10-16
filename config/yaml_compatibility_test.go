package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigYAMLRoundTrip(t *testing.T) {
	fixtures, err := filepath.Glob("mock/*.yaml")
	if err != nil || len(fixtures) == 0 {
		t.Fatalf("find legacy fixtures: %v", err)
	}
	fixtures = append(fixtures, "testdata/compatibility.yaml")
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			original := Config{Path: fixture}
			if err := original.Load(); err != nil {
				t.Fatal(err)
			}
			original.Path = filepath.Join(t.TempDir(), "saved.yaml")
			if err := original.Save(); err != nil {
				t.Fatal(err)
			}
			reloaded := Config{Path: original.Path}
			if err := reloaded.Load(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(original.HTTP, reloaded.HTTP) {
				t.Fatalf("configuration changed after save/load:\nbefore: %#v\nafter: %#v", original.HTTP, reloaded.HTTP)
			}
		})
	}
}

func TestConfigYAMLCompatibilityFields(t *testing.T) {
	c := Config{Path: "testdata/compatibility.yaml"}
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	https := c.HTTP.Routers["compatibility-https"]
	if https == nil || https.Rule != "Host(`compat.example.com`)" || https.Service != "compatibility" ||
		!reflect.DeepEqual(https.Entrypoints, []string{"websecure"}) ||
		https.TLS == nil || https.TLS.CertResolver != "http01" ||
		!reflect.DeepEqual(https.Middlewares, []string{"headers", "basic", "restricted", "forward"}) {
		t.Fatalf("HTTPS routing/TLS/middleware settings changed: %#v", https)
	}
	http := c.HTTP.Routers["compatibility-http"]
	if http == nil || !reflect.DeepEqual(http.Entrypoints, []string{"web"}) || http.TLS != nil ||
		!reflect.DeepEqual(http.Middlewares, []string{"redirect", "headers"}) {
		t.Fatalf("HTTP routing settings changed: %#v", http)
	}
	service := c.HTTP.Services["compatibility"]
	if service == nil || !reflect.DeepEqual(service.LoadBalancer.Servers, []server{
		{URL: "http://127.0.0.1:8080/path?mode=on&limit=0123"},
		{URL: "https://backend.example.com:8443"},
	}) {
		t.Fatalf("backend addresses changed: %#v", service)
	}
	// These are strings, even when their spelling resembles YAML booleans/numbers.
	headers := c.HTTP.Middlewares["headers"].Headers
	if !reflect.DeepEqual(headers.CustomRequestHeaders, map[string]string{
		"X-Mode": "on", "X-Enabled": "yes", "X-Code": "0123", "X-Unicode": "Grüezi",
	}) || headers.STSSeconds != 31536000 {
		t.Fatalf("header values changed: %#v", headers)
	}
	basic := c.HTTP.Middlewares["basic"].BasicAuth
	if !reflect.DeepEqual(basic.Users, []string{"alice:$2a$10$synthetic-test-hash"}) ||
		basic.Realm != "Private: area" || basic.HeaderField != "X-Authenticated-User" || !basic.RemoveHeader {
		t.Fatalf("basic auth settings changed: %#v", basic)
	}
	ip := c.HTTP.Middlewares["restricted"].IPWhiteList
	if !reflect.DeepEqual(ip.SourceRange, []string{"192.0.2.0/24", "2001:db8::/32"}) ||
		ip.IPStrategy == nil || ip.IPStrategy.Depth != 2 {
		t.Fatalf("IP restrictions changed: %#v", ip)
	}
	redirect := c.HTTP.Middlewares["redirect"].RedirectScheme
	if redirect.Scheme != "https" || !redirect.Permanent ||
		c.HTTP.Middlewares["forward"].ForwardAuth.Address != "http://127.0.0.1:8099/auth" {
		t.Fatal("redirect or forward auth settings changed")
	}
}

func TestConfigLoadRejectsMalformedYAML(t *testing.T) {
	for name, contents := range map[string]string{
		"syntax":      "http: [\n",
		"router type": "http:\n  routers: not-a-map\n",
		"TLS type":    "http:\n  routers:\n    test:\n      tls: [invalid]\n",
	} {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "invalid.yaml")
			if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			c := Config{Path: file}
			if err := c.Load(); err == nil || c.loaded {
				t.Fatalf("malformed file accepted or marked loaded: %v", err)
			}
		})
	}
}

func TestConfigLoadDuplicateKeyCompatibility(t *testing.T) {
	// yaml.v2 accepts duplicate keys and keeps the last value. A migration to a
	// stricter parser must make that compatibility change explicit.
	file := filepath.Join(t.TempDir(), "duplicate.yaml")
	contents := "http:\n  routers:\n    test:\n      rule: first\n      rule: second\n"
	if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	c := Config{Path: file}
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	if c.HTTP.Routers["test"].Rule != "second" {
		t.Fatal("duplicate-key handling changed")
	}
}
