package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Set TRAEFIK_BINARY to exercise the same fixtures against a downloaded release.
// CI takes the version from Dockerfile so dependency bumps run this test too.
func TestTraefikFixtures(t *testing.T) {
	binary := os.Getenv("TRAEFIK_BINARY")
	if binary == "" {
		t.Skip("set TRAEFIK_BINARY to run real Traefik fixture checks")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"path": r.URL.RequestURI(), "mode": r.Header.Get("X-Mode"),
			"enabled": r.Header.Get("X-Enabled"), "code": r.Header.Get("X-Code"),
			"unicode": r.Header.Get("X-Unicode"), "authorization": r.Header.Get("Authorization"),
		})
	}))
	t.Cleanup(backend.Close)
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Regression-Auth") != "allow" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(auth.Close)

	root := t.TempDir()
	dynamic := filepath.Join(root, "dynamic")
	if err := os.Mkdir(dynamic, 0700); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("testdata/traefik/*.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("find runtime fixtures: %v", err)
	}
	for _, file := range files {
		c := Config{Path: file}
		if err := c.Load(); err != nil {
			t.Fatal(err)
		}
		for _, service := range c.HTTP.Services {
			for i := range service.LoadBalancer.Servers {
				service.LoadBalancer.Servers[i].URL = backend.URL
			}
		}
		if middleware := c.HTTP.Middlewares["sys-forwardauth"]; middleware != nil {
			middleware.ForwardAuth.Address = auth.URL
		}
		c.Path = filepath.Join(dynamic, filepath.Base(file))
		if err := c.Save(); err != nil {
			t.Fatal(err)
		}
	}

	web, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = web.Close() })
	secure, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = secure.Close() })
	webAddress, secureAddress := web.Addr().String(), secure.Addr().String()
	_ = web.Close()
	_ = secure.Close()
	logFile, err := os.Create(filepath.Join(root, "traefik.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logFile.Close() })
	command := exec.Command(binary,
		"--global.checknewversion=false", "--global.sendanonymoususage=false",
		"--log.level=ERROR", "--providers.file.directory="+dynamic, "--providers.file.watch=false",
		"--entrypoints.web.address="+webAddress, "--entrypoints.websecure.address="+secureAddress)
	command.Dir, command.Stdout, command.Stderr = root, logFile, logFile
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_ = command.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		_ = command.Process.Kill()
		<-exited
		if t.Failed() {
			data, _ := os.ReadFile(logFile.Name())
			t.Logf("Traefik output:\n%s", data)
		}
	})
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} // Local, generated test certificate only.
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request := func(host string, https, authenticated, forwardAllowed bool) (*http.Response, error) {
		base := "http://" + webAddress
		if https {
			base = "https://" + secureAddress
		}
		r, err := http.NewRequest(http.MethodGet, base+"/fixture?value=a%2Fb", nil)
		if err != nil {
			return nil, err
		}
		r.Host = host
		if authenticated {
			r.SetBasicAuth("test", "password")
		}
		if forwardAllowed {
			r.Header.Set("X-Regression-Auth", "allow")
		}
		return client.Do(r)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		select {
		case <-exited:
			t.Fatal("Traefik exited before fixtures became ready")
		default:
		}
		response, err := request("plain.example.test", false, false, false)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("Traefik did not load fixture routes within 20 seconds")
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, fixture := range fixtureCases(t) {
		t.Run(fixture.File, func(t *testing.T) {
			response, err := request(fixture.Domain, fixture.HTTPS, true, true)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("route returned %d", response.StatusCode)
			}
			var body map[string]string
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["path"] != "/fixture?value=a%2Fb" {
				t.Fatalf("request path changed: %v", body)
			}
			if fixture.File == "Headers_20000002.yaml" && (body["mode"] != "on" || body["enabled"] != "yes" ||
				body["code"] != "0123" || body["unicode"] != "Grüezi") {
				t.Fatalf("header middleware changed scalar values: %v", body)
			}
			if fixture.HSTS && response.Header.Get("Strict-Transport-Security") != "max-age=31536000" {
				t.Fatal("HSTS header missing or changed")
			}
			if fixture.BasicAuthUser != "" && body["authorization"] != "" {
				t.Fatal("basic auth credential reached the backend despite removeHeader")
			}
			for _, denied := range []struct {
				enabled bool
				status  int
			}{{fixture.BasicAuthUser != "", http.StatusUnauthorized}, {fixture.ForwardAuth, http.StatusForbidden}} {
				if denied.enabled {
					r, err := request(fixture.Domain, fixture.HTTPS, false, false)
					if err != nil {
						t.Fatal(err)
					}
					_ = r.Body.Close()
					if r.StatusCode != denied.status {
						t.Fatalf("unauthenticated route returned %d, want %d", r.StatusCode, denied.status)
					}
				}
			}
			if fixture.ForceTLS {
				r, err := request(fixture.Domain, false, false, false)
				if err != nil {
					t.Fatal(err)
				}
				_ = r.Body.Close()
				if r.StatusCode != http.StatusMovedPermanently || r.Header.Get("Location") != fmt.Sprintf("https://%s/fixture?value=a%%2Fb", fixture.Domain) {
					t.Fatalf("HTTPS redirect changed: %d %q", r.StatusCode, r.Header.Get("Location"))
				}
			}
		})
	}
}
