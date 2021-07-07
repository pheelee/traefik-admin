package config

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

var sample1 Config = Config{
	Path: "",
	HTTP: HTTP{
		Routers: map[string]*Router{
			"Test": {
				Entrypoints: []string{
					"websecure",
				},
				Rule:    "Host(`test.example.com`)",
				Service: "Test",
				TLS: &routerTLSConfig{
					CertResolver: "http01",
				},
				Middlewares: []string{
					"sys-hsts@file",
					"Test-headers",
				},
			},
			"Test-http": {
				Entrypoints: []string{
					"web",
				},
				Rule:    "Host(`test.example.com`)",
				Service: "Test",
				Middlewares: []string{
					"sys-redirscheme@file",
					"Test-headers",
				},
			},
		},
		Services: map[string]*Service{
			"Test": {
				LoadBalancer: loadbalancer{
					Servers: []server{
						{
							URL: "http://1.2.3.4:80",
						},
					},
				},
			},
		},
		Middlewares: map[string]*Middleware{
			"Test-headers": {
				Headers: Headers{
					CustomRequestHeaders: map[string]string{
						"X-Server-IP": "$ServerIP",
					},
				},
			},
		},
	},
}

var ui UserInput = UserInput{
	Name:   "Test",
	Domain: "test.example.com",
	Backend: Backend{
		URL: "http://1.2.3.4:80",
	},
	ForwardAuth: true,
	HTTPS:       true,
	ForceTLS:    true,
	HSTS:        true,
	Headers: []headersInput{
		{
			Name:  "X-Test-Header",
			Value: "TestValue",
		},
		{
			Name:  "X-Server-IP",
			Value: "$ServerIP",
		},
	},
	BasicAuth: []basicAuthInput{
		{
			Username: "test",
			Password: "123456",
		},
	},
	IPRestriction: &ipRestriction{
		Depth: 1,
		IPs: []string{
			"192.168.1.0/24",
		},
	},
}

func TestLoad(t *testing.T) {
	d, err := os.ReadDir("./mock")
	if err != nil {
		t.Error("Could not read mock directory")
	}
	for _, f := range d {
		c := Config{Path: path.Join("./mock", f.Name())}
		if !f.IsDir() {
			err := c.Load()
			if err != nil {
				t.Errorf("Failed load config %s with error %s", f.Name(), err)
			}
		}
	}

	c := Config{Path: "/tmp/wefiuhweigaergeargufh"}
	if c.Load() == nil {
		t.Errorf("Config /tmp/wefiuhweigufh should fail")
	}

	c = Config{loaded: true}
	if c.Load() != nil {
		t.Errorf("Config loaded should yield nil")
	}
}

func TestName(t *testing.T) {
	cases := []struct{ Path, Name string }{
		{Path: "./dynamic.d/Test_12345678.yaml", Name: "Test"},
		{Path: "./dynamic.d/Test.yaml", Name: "Test"},
		{Path: "./dynamic.d/Test_12345678", Name: "Test"},
	}

	for _, cs := range cases {
		c := Config{Path: cs.Path}
		name := c.Name()
		if name != cs.Name {
			t.Errorf("%s != %s", name, cs.Name)
		}
	}
}

func TestID(t *testing.T) {
	cases := []struct{ Path, ID string }{
		{Path: "./dynamic.d/Test_12345678.yaml", ID: "Test_12345678"},
		{Path: "./dynamic.d/Test.yaml", ID: "Test"},
		{Path: "./dynamic.d/Test_12345678", ID: "Test_12345678"},
	}

	for _, cs := range cases {
		c := Config{Path: cs.Path}
		ID := c.ID()
		if ID != cs.ID {
			t.Errorf("%s != %s", ID, cs.ID)
		}
	}
}

func TestContainsRouter(t *testing.T) {

	if sample1.HTTP.containsRouter("Test") == false {
		t.Errorf("Should contain router Test")
	}

	if sample1.HTTP.containsRouter("Test2") == true {
		t.Errorf("Should not contain router Test2")
	}

}

func TestToUserInput(t *testing.T) {
	d, err := os.ReadDir("./mock")
	if err != nil {
		t.Error("Could not read mock directory")
	}
	for _, f := range d {
		c := Config{Path: path.Join("./mock", f.Name())}
		if !f.IsDir() {
			_, err := c.ToUserInput()
			if err != nil {
				t.Errorf("%s should pass ToUserInput", c.Path)
			}
		}
	}

	c := Config{Path: "/tmp/wsekfubwsieufghi7g"}
	_, err = c.ToUserInput()
	if err == nil {
		t.Errorf("Should fail because config does not exist")
	}
}

func TestSave(t *testing.T) {
	sample1.Path = "/root212414/Sample1_12345678.yaml"
	if sample1.Save() == nil {
		t.Error("Should return error")
	}
	tmp, _ := ioutil.TempFile("", "")
	defer os.Remove(tmp.Name())
	sample1.Path = tmp.Name()
	if sample1.Save() != nil {
		t.Error("Should return nil")
	}
}

func TestChangeIdentifier(t *testing.T) {
	tmp, _ := ioutil.TempFile("", "")
	defer os.Remove(tmp.Name())
	sample1.Path = tmp.Name()
	sample1.Save()
	sample1.ChangeIdentifier("Test", "Test_654321")
}

func TestFromUserInput(t *testing.T) {
	c := FromUserInput(&ui, "http01")
	if c == nil {
		t.Error("Should be non nil")
	}
	c = FromUserInput(&UserInput{
		Name:        "Test",
		ForwardAuth: true,
	}, "http01")
	if c != nil {
		t.Error("Should be nil")
	}
}

func TestRandHash(t *testing.T) {
	if RandHash() == "" {
		t.Error("Should return a string")
	}
}

func TestSpliceEmpty(t *testing.T) {
	var t1 []string = []string{"One", "Two", "", "Four"}
	r := spliceEmpty(t1)
	if !reflect.DeepEqual(r, []string{"One", "Two", "Four"}) {
		t.Errorf("Wrong slice %s", r)
	}
}
