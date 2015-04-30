package main

import (
	"flag"
	"os"

	"github.com/tehmaze-labs/dns/pdns"
)

func main() {
	var config string

	flag.StringVar(&config, "config", "testdata/dns.json", "configuration file")
	flag.Parse()

	c, err := NewConfig(config)
	if err != nil {
		panic(err)
	}

	r, err := c.Resolvers()
	if err != nil {
		panic(err)
	}

	p := pdns.New(r)
	p.Serve(os.Stdin, os.Stdout)
}
