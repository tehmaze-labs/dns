package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tehmaze-labs/dns/config"
)

func main() {
	var filename string

	flag.StringVar(&filename, "config", "testdata/dns.yaml", "configuration file")
	flag.Parse()

	c, err := config.NewConfig(filename)
	if err != nil {
		fmt.Printf("error parsing %q: %v\n", filename, err)
		os.Exit(1)
	}

	r, err := c.Backends()
	if err != nil {
		panic(err)
	}

	p := New(r)
	p.Serve(os.Stdin, os.Stdout)
}
