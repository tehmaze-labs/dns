package main

import "testing"

func TestNewConfig(t *testing.T) {
	c, err := NewConfig("testdata/dns.yaml")
	if err != nil {
		t.Error(err)
		return
	}

	r, err := c.Resolvers()
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("got %d resolvers", len(r))
	for k, v := range r {
		t.Logf("resolver %q: %v", k, v)
	}
}
