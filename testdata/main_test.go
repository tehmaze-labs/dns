package main

import (
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v2"
)

var filename = "dns.yaml"

type Config struct {
	Resolver []yaml.MapItem
}

func testParse(t *testing.T) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Error(err)
		return
	}

	c := Config{}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		t.Error(err)
		return
	}
}
