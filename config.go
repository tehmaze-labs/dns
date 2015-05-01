package main

import (
	"errors"
	"io/ioutil"

	"github.com/tehmaze-labs/dns/backend"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Backend   *backend.BackendConfig `yaml:"backend"`
	Templates interface{}            `yaml:"templates"`
}

func NewConfig(filename string) (c *Config, err error) {
	var data []byte

	if data, err = ioutil.ReadFile(filename); err != nil {
		return nil, err
	}

	c = &Config{}
	if err = yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}

	return
}

func (c *Config) Backends() (bs []backend.Backend, err error) {
	bs = make([]backend.Backend, 0)

	for _, b := range c.Backend.AutoBackends {
		if err = b.Check(); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	for _, b := range c.Backend.GeoBackends {
		if err = b.Check(); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}

	if len(bs) == 0 {
		return nil, errors.New("no backends configured")
	}

	return
}
