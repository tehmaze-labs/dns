package config

import "testing"

func TestNewConfig(t *testing.T) {
	c, err := NewConfig("testdata/dns.yaml")
	if err != nil {
		t.Error(err)
		return
	}

	r, err := c.Backends()
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("got %d backends", len(r))
	for k, v := range r {
		t.Logf("backend %q: %v", k, v)
	}
}
