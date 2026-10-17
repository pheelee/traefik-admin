package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMultipleBackendValidation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		urls       []string
		valid      bool
		errorIndex int
	}{
		{"nil", nil, false, 0},
		{"empty", []string{"", ""}, false, 0},
		{"one", []string{"http://127.0.0.1:8080"}, true, 0},
		{"gaps", []string{"", "http://127.0.0.1:8080", ""}, true, 0},
		{"invalid-second", []string{"http://127.0.0.1:8080", "not-a-url"}, false, 1},
		{"five", []string{"http://127.0.0.1:1", "http://127.0.0.1:2", "http://127.0.0.1:3", "http://127.0.0.1:4", "http://127.0.0.1:5"}, true, 0},
		{"six", []string{"http://127.0.0.1:1", "http://127.0.0.1:2", "http://127.0.0.1:3", "http://127.0.0.1:4", "http://127.0.0.1:5", "http://127.0.0.1:6"}, false, 0},
		{"invalid-sixth", []string{"http://127.0.0.1:1", "http://127.0.0.1:2", "http://127.0.0.1:3", "http://127.0.0.1:4", "http://127.0.0.1:5", "not-a-url"}, false, 5},
		{"invalid-after-gaps", []string{"", "", "", "", "", "not-a-url"}, false, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := &UserInput{Name: "Validation", Domain: "test.example.com"}
			for _, url := range tc.urls {
				u.Backends = append(u.Backends, Backend{URL: url})
			}
			v := u.Validate()
			if v.Valid != tc.valid {
				t.Fatalf("valid = %v, want %v: %+v", v.Valid, tc.valid, v.Errors)
			}
			if !tc.valid && v.Errors.Backends[tc.errorIndex] == "" {
				t.Fatalf("missing error at backend %d", tc.errorIndex)
			}
			if !tc.valid && FromUserInput(u, "") != nil {
				t.Fatal("invalid backends generated a config")
			}
		})
	}
}

func TestMultipleBackendRoundTrip(t *testing.T) {
	for _, count := range []int{1, 2, MaxBackends} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			m := &ConfigManager{Path: t.TempDir()}
			u := &UserInput{Name: "RoundTrip", Domain: "test.example.com"}
			for i := 0; i < count; i++ {
				u.Backends = append(u.Backends, Backend{URL: fmt.Sprintf("http://127.0.0.1:%d", 18080+i)})
			}
			c, err := m.Add(u)
			if err != nil {
				t.Fatal(err)
			}
			read, err := c.ToUserInput()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(read.Backends[:count], u.Backends) {
				t.Fatalf("backends changed: %+v", read.Backends)
			}
			read.Domain = "updated.example.com"
			updated, err := m.Update(read)
			if err != nil {
				t.Fatal(err)
			}
			if err := updated.Load(); err != nil {
				t.Fatal(err)
			}
			servers := updated.HTTP.Services[updated.ID()].LoadBalancer.Servers
			if len(servers) != count {
				t.Fatalf("got %d servers, want %d", len(servers), count)
			}
			for i, server := range servers {
				if server.URL != u.Backends[i].URL {
					t.Fatalf("backend %d changed", i)
				}
			}
		})
	}
}

func TestOversizedExistingConfigIsPreserved(t *testing.T) {
	m := &ConfigManager{Path: t.TempDir()}
	u := &UserInput{Name: "Existing", Domain: "existing.example.com", Backends: []Backend{{URL: "http://127.0.0.1:8080"}}}
	c, err := m.Add(u)
	if err != nil {
		t.Fatal(err)
	}
	servers := &c.HTTP.Services[c.id].LoadBalancer.Servers
	for i := 1; i < 6; i++ {
		*servers = append(*servers, server{URL: fmt.Sprintf("http://127.0.0.1:%d", 8080+i)})
	}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := c.ToUserInput()
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Backends) != 6 {
		t.Fatalf("existing servers truncated to %d", len(read.Backends))
	}
	if _, err := m.Update(read); err == nil {
		t.Fatal("oversized update accepted")
	}
	after, err := os.ReadFile(c.Path)
	if err != nil || string(after) != string(original) {
		t.Fatalf("rejected update modified original: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(m.Path, "*.yaml"))
	if err != nil || len(files) != 1 {
		t.Fatalf("rejected update changed files: %v", err)
	}
	// The editor can explicitly clear surplus rows before saving.
	read.Backends[5].URL = ""
	updated, err := m.Update(read)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.HTTP.Services[updated.id].LoadBalancer.Servers) != MaxBackends {
		t.Fatal("cleared backend was written")
	}
}
