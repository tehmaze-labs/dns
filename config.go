package main

import (
	"errors"
	"io/ioutil"

	"github.com/tehmaze-labs/dns/resolver"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Resolver  *resolver.ResolverConfig `yaml:"resolver"`
	Templates interface{}              `yaml:"templates"`
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

func (c *Config) Resolvers() (rs []resolver.Resolver, err error) {
	rs = make([]resolver.Resolver, 0)

	for _, r := range c.Resolver.AutoResolvers {
		rs = append(rs, r)
	}

	if len(rs) == 0 {
		return nil, errors.New("no resolvers configured")
	}

	return
}
