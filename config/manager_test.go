package config

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

var oldConfig string = `
http:
  routers:
    Test:
      entryPoints:
      - websecure
      rule: Host(` + "`" + `test.example.com` + "`" + `)
      service: Test
      tls:
        certResolver: http01
    Test-http:
      entryPoints:
      - web
      rule: Host(` + "`" + `test.example.com` + "`" + `)
      service: Test
      middlewares:
      - sys-redirscheme@file
  services:
    Test:
      loadBalancer:
        servers:
        - url: https://192.168.1.5:8006

`

func setupError(t *testing.T) ConfigManager {
	M := ConfigManager{Path: "/roiwjegoijwerg", CertResolver: "http01"}
	M.Add(&ui)
	ui.Name = "Test2"
	M.Add(&ui)
	ui.Name = "Test3"
	M.Add(&ui)
	return M
}

func setupSuccess(t *testing.T) ConfigManager {
	M := ConfigManager{Path: t.TempDir(), CertResolver: "http01"}
	M.Add(&ui)
	ui.Name = "Test2"
	M.Add(&ui)
	ui.Name = "Test3"
	M.Add(&ui)
	return M
}

func TestAdd(t *testing.T) {
	M := ConfigManager{Path: t.TempDir(), CertResolver: "http01"}
	_, err := M.Add(&ui)
	if err != nil {
		t.Error("Should be nil")
	}

	M = ConfigManager{Path: "/rootiowjeifgjweiogjwieg/aergerg", CertResolver: "http01"}
	_, err = M.Add(&ui)
	if err == nil {
		t.Error("Should be non nil")
	}

	// Test arbitrary data
	_, err = M.Add(&UserInput{
		Name:        "Test",
		Domain:      "sdrgedrhg",
		ForwardAuth: true,
	})
	if err == nil {
		t.Error("Should be non nil")
	}
}

func TestUpdate(t *testing.T) {
	M := ConfigManager{Path: t.TempDir(), CertResolver: "http01"}
	c, _ := M.Add(&ui)
	ui.Name = "Test2"
	ui.ID = c.id
	_, err := M.Update(&ui)
	if err != nil {
		t.Errorf("Should be nil, got %s", err)
	}

	ui.ID = "123"
	_, err = M.Update(&ui)
	if err == nil {
		t.Error("Should be non nil")
	}
}

func TestListUserInputs(t *testing.T) {
	M := setupSuccess(t)

	l, err := M.ListUserInputs()
	if err != nil || len(l) != 3 {
		t.Error("Should be nil and len=3")
	}
	M = setupError(t)
	l, err = M.ListUserInputs()
	if err == nil && len(l) == 3 {
		t.Error("Should be non nil and len=3")
	}
}

func TestSetCertResolver(t *testing.T) {
	M := setupSuccess(t)

	err := M.SetCertResolver("dns01")
	if err != nil {
		t.Error("Should be nil")
	}
	cl, _ := M.List()
	for _, c := range cl {
		c.Load()
		if c.HTTP.Routers[c.ID()].TLS.CertResolver != "dns01" {
			t.Errorf("CertResolver not switched")
		}
	}
}

func TestSetForwardAuth(t *testing.T) {
	M := ConfigManager{Path: t.TempDir(), CertResolver: "http01"}
	c, _ := M.Add(&UserInput{
		Name:        "Test",
		Domain:      "test.example.com",
		Backend:     Backend{URL: "http://1.2.3.4:80"},
		ForwardAuth: true,
	})
	err := M.SetForwardAuth(Remove)
	if err != nil {
		t.Error("Should be nil")
	}
	c = M.Get(c.id)
	if c.HTTP.hasAnyRouterMiddleware(FORWARDAUTH) {
		t.Error("No router should have forward auth middleware")
	}

}

func TestMigrateConfig(t *testing.T) {
	temp := t.TempDir()
	ioutil.WriteFile(path.Join(temp, "Test.yaml"), []byte(oldConfig), 0644)
	M := ConfigManager{Path: temp, CertResolver: "http01"}
	err := M.MigrateConfig()
	if err != nil {
		t.Error(err)
	}
	if _, err = os.Stat(path.Join(temp, "Test.yaml")); err == nil {
		t.Error("config not renamed")
	}
}

func TestSplice(t *testing.T) {
	s := []string{"One", "Two", "Three", "Four"}
	if !reflect.DeepEqual(splice(s, "Two"), []string{"One", "Three", "Four"}) {
		t.Error("slice mismatch")
	}
}
