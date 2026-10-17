package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fixtureCase struct {
	Backends      []string `json:"backends"`
	File          string   `json:"file"`
	Domain        string   `json:"domain"`
	HTTPS         bool     `json:"https"`
	ForceTLS      bool     `json:"forcetls"`
	HSTS          bool     `json:"hsts"`
	ForwardAuth   bool     `json:"forwardauth"`
	BasicAuthUser string   `json:"basicAuthUser"`
}

func fixtureCases(t *testing.T) []fixtureCase {
	t.Helper()
	data, err := os.ReadFile("testdata/traefik/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []fixtureCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("fixture manifest is empty")
	}
	return cases
}

func TestTraefikFixtureUserInputs(t *testing.T) {
	for _, fixture := range fixtureCases(t) {
		t.Run(fixture.File, func(t *testing.T) {
			c := Config{Path: filepath.Join("testdata/traefik", fixture.File)}
			u, err := c.ToUserInput()
			if err != nil {
				t.Fatal(err)
			}
			expectedBackends := fixture.Backends
			if len(expectedBackends) == 0 {
				expectedBackends = []string{"http://127.0.0.1:18080"}
			}
			var actualBackends []string
			for _, b := range u.Backends {
				if b.URL != "" {
					actualBackends = append(actualBackends, b.URL)
				}
			}
			if !reflect.DeepEqual(actualBackends, expectedBackends) {
				t.Fatalf("backend addresses changed: %v", actualBackends)
			}
			if u.ID != strings.TrimSuffix(fixture.File, ".yaml") || u.Domain != fixture.Domain ||
				u.Backends[0].URL != "http://127.0.0.1:18080" || u.HTTPS != fixture.HTTPS ||
				u.ForceTLS != fixture.ForceTLS || u.HSTS != fixture.HSTS || u.ForwardAuth != fixture.ForwardAuth ||
				u.BasicAuth[0].Username != fixture.BasicAuthUser {
				t.Fatalf("fixture does not produce its expected UI settings: %+v", u)
			}
		})
	}
}

func TestInvalidFixturesAreNotRewrittenByManager(t *testing.T) {
	files, err := filepath.Glob("testdata/invalid/*.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("find invalid fixtures: %v", err)
	}
	operations := map[string]func(*ConfigManager) error{
		"migrate":  (*ConfigManager).MigrateConfig,
		"resolver": func(m *ConfigManager) error { return m.SetCertResolver("changed") },
		"auth":     func(m *ConfigManager) error { return m.SetForwardAuth(Remove) },
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for name, operation := range operations {
			t.Run(filepath.Base(file)+"/"+name, func(t *testing.T) {
				directory := t.TempDir()
				path := filepath.Join(directory, "Broken.yaml")
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
				err := operation(&ConfigManager{Path: directory})
				if err == nil || !strings.Contains(err.Error(), path) {
					t.Fatalf("expected an error naming the invalid file, got %v", err)
				}
				after, err := os.ReadFile(path)
				if err != nil || string(after) != string(data) {
					t.Fatalf("invalid file was moved or rewritten: %v", err)
				}
				entries, err := os.ReadDir(directory)
				if err != nil || len(entries) != 1 {
					t.Fatal("manager wrote additional files after rejecting invalid input")
				}
			})
		}
	}
}
