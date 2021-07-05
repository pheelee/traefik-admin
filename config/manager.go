package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

var Manager ConfigManager

type ConfigManager struct {
	Path         string
	CertResolver string
}

type Operation int

const (
	Add Operation = iota
	Remove
)

func (m *ConfigManager) Add(u *UserInput) (*Config, error) {
	// Generate Config
	c := FromUserInput(u, m.CertResolver)
	// Set Path
	c.Path = path.Join(m.Path, c.id+".yaml")
	if err := c.Save(); err != nil {
		return nil, err
	}
	return c, nil
}

//Update must search a config by hash, delete it and write the new config to file
func (m *ConfigManager) Update(u *UserInput) (*Config, error) {
	if err := m.Delete(u.ID); err != nil {
		return nil, err
	}
	return m.Add(u)
	/*
		n := strings.Replace(u.ID, path.Ext(u.ID), "", -1)
		hash := strings.Split(n, "_")[1]
		c := m.GetByHash(hash)
		if c == nil {
			return nil, fmt.Errorf("config not found")
		}
		if err := os.Remove(c.Path); err != nil {
			return nil, err
		}
		return m.Add(u)*/
}

func (m *ConfigManager) Delete(id string) error {
	c := m.Get(id)
	if c == nil {
		return fmt.Errorf("config not found")
	}
	return os.Remove(c.Path)
}

func (m *ConfigManager) Get(id string) *Config {
	cl, err := Manager.List()
	if err != nil {
		return nil
	}
	for _, c := range cl {
		if c.ID() == id {
			if err := c.Load(); err != nil {
				return nil
			}
			return &c
		}
	}
	return nil
}

func (m *ConfigManager) List() ([]Config, error) {
	var (
		cl  []Config = make([]Config, 0)
		err error
		fi  []os.FileInfo
	)

	if fi, err = ioutil.ReadDir(m.Path); err != nil {
		return nil, err
	}

	for _, f := range fi {
		if path.Ext(f.Name()) == ".yaml" && !strings.HasPrefix(f.Name(), "sys_") {
			cl = append(cl, Config{Path: path.Join(m.Path, f.Name())})
		}
	}
	return cl, nil
}

func (m *ConfigManager) ListUserInputs() ([]UserInput, error) {
	uil := []UserInput{}
	cl, err := m.List()
	if err != nil {
		return uil, err
	}
	for _, c := range cl {
		u, err := c.ToUserInput()
		if err != nil {
			return uil, err
		}
		uil = append(uil, *u)
	}
	return uil, nil
}

func (m *ConfigManager) SetCertResolver(r string) error {
	cl, err := m.List()
	if err != nil {
		return err
	}
	for _, c := range cl {
		for k, e := range c.HTTP.Routers {
			if e.TLS != nil {
				c.HTTP.Routers[k].TLS.CertResolver = r
				if err := c.Save(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (m *ConfigManager) SetForwardAuth(o Operation) error {
	cl, err := m.List()
	if err != nil {
		return err
	}
	for _, c := range cl {
		if err := c.Load(); err != nil {
			return err
		}
		for _, r := range c.HTTP.Routers {
			switch o {
			case Add:
				if !r.hasMiddleware(FORWARDAUTH) {
					r.Middlewares = append(r.Middlewares, FORWARDAUTH)
				}
			case Remove:
				r.Middlewares = splice(r.Middlewares, FORWARDAUTH)
			}
		}
		if err := c.Save(); err != nil {
			return err
		}
	}
	return nil
}

func (m *ConfigManager) MigrateConfig() error {
	cl, err := m.List()
	if err != nil {
		return err
	}
	for _, c := range cl {
		id := c.ID()
		p := strings.Split(id, "_")
		if len(p) < 2 {
			nn := id + "_" + RandHash()
			c.ChangeIdentifier(id, nn)
			if err := os.Rename(c.Path, strings.Replace(c.Path, id, nn, -1)); err != nil {
				return err
			}
		}
	}
	return nil
}

func splice(slice []string, needle string) []string {
	s := []string{}
	for _, i := range slice {
		if i != needle {
			s = append(s, i)
		}
	}
	return s
}
