package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tehmaze-labs/dns/pdns"
)

func main() {
	var config string

	flag.StringVar(&config, "config", "testdata/dns.yaml", "configuration file")
	flag.Parse()

	c, err := NewConfig(config)
	if err != nil {
		fmt.Printf("error parsing %q: %v\n", config, err)
		os.Exit(1)
	}

	r, err := c.Backends()
	if err != nil {
		panic(err)
	}

	p := pdns.New(r)
	p.Serve(os.Stdin, os.Stdout)
}
