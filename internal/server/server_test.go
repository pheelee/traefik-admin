package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pheelee/traefik-admin/config"
)

func TestMultipleBackendAPI(t *testing.T) {
	previous := config.Manager
	t.Cleanup(func() { config.Manager = previous })
	config.Manager = config.ConfigManager{Path: t.TempDir()}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(backend.Close)
	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	handler := SetupRoutes(Config{})
	request := func(method, path string, body any, status int) *httptest.ResponseRecorder {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	decode := func(w *httptest.ResponseRecorder) config.UserInput {
		t.Helper()
		var u config.UserInput
		if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
			t.Fatal(err)
		}
		return u
	}
	u := config.UserInput{Name: "APITest", Domain: "api.example.com", Backends: []config.Backend{{URL: backend.URL}, {URL: dead.URL}}}
	created := decode(request(http.MethodPost, "/config/new", u, http.StatusOK))
	if !created.Backends[0].Healthy || created.Backends[1].Healthy {
		t.Fatalf("incorrect independent health: %+v", created.Backends)
	}
	list := request(http.MethodGet, "/config/", nil, http.StatusOK)
	var entries []config.UserInput
	if err := json.Unmarshal(list.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].Backends[0].Healthy || entries[0].Backends[1].Healthy {
		t.Fatalf("incorrect list: %s", list.Body.String())
	}
	file := filepath.Join(config.Manager.Path, created.ID+".yaml")
	original, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	tooMany := created
	tooMany.Backends = []config.Backend{}
	for i := 0; i < 6; i++ {
		tooMany.Backends = append(tooMany.Backends, config.Backend{URL: backend.URL})
	}
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		request(method, "/config/"+created.ID, tooMany, http.StatusBadRequest)
		tooMany.Backends[5].URL = "not-a-url"
		request(method, "/config/"+created.ID, tooMany, http.StatusBadRequest)
		tooMany.Backends[5].URL = backend.URL
	}
	after, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatalf("invalid request changed original: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(config.Manager.Path, "*.yaml"))
	if err != nil || len(files) != 1 {
		t.Fatalf("invalid request created files: %v", err)
	}
	created.Backends[1].URL = ""
	updated := decode(request(http.MethodPut, "/config/"+created.ID, created, http.StatusOK))
	if updated.Backends[0].URL != backend.URL || updated.Backends[1].URL != "" {
		t.Fatalf("backend removal failed: %+v", updated.Backends)
	}
	request(http.MethodDelete, "/config/"+updated.ID, nil, http.StatusOK)
}
