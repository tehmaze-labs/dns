package resolver

import (
	"net"

	"github.com/miekg/dns"
	"github.com/tehmaze-labs/dns/encoder"
	"github.com/tehmaze-labs/dns/message"
)

var typesA = map[uint16]bool{
	dns.TypeANY: true,
	dns.TypeA:   true,
}
var typesAAAA = map[uint16]bool{
	dns.TypeANY:  true,
	dns.TypeAAAA: true,
}
var typesPTR = map[uint16]bool{
	dns.TypeANY: true,
	dns.TypePTR: true,
}

type AutoResolver struct {
	Encode         string `yaml:"encode"`
	Filler         bool   `yaml:"filler"`
	Prefix, Suffix string
	SOA            string
	DNS            []string
	Answers        map[string]*AutoResolverAnswer

	encoder encoder.Encoder
}

type AutoResolverAnswer struct {
	Network        *net.IPNet
	Size           int
	Zone           string
	Prefix, Suffix string
	Version        uint8
}

func (r *AutoResolver) Check() (err error) {
	r.encoder, err = encoder.NewEncoder(r.Encode)
	if err != nil {
		return
	}

	for zone, answer := range r.Answers {
		_, answer.Network, err = net.ParseCIDR(zone)
		if err != nil {
			return err
		}
		if answer.Size == 0 {
			ones, _ := answer.Network.Mask.Size()
			answer.Size = ones
		}
		if answer.Prefix == "" && r.Prefix != "" {
			answer.Prefix = r.Prefix
		}
		if answer.Suffix == "" && r.Suffix != "" {
			answer.Suffix = r.Suffix
		}
	}

	return
}

func (r *AutoResolver) Query(m *message.Message) ([]*message.Message, error) {
	return nil, nil
}
