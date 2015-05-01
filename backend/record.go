package backend

import (
	"fmt"

	"github.com/miekg/dns"
	"github.com/tehmaze-labs/dns/message"
)

type Record struct {
	Class   string `json:"class"`
	Type    string `json:"type"`
	TTL     int    `json:"ttl"`
	Content string `json:"content"`
}

func (r *Record) Message() (*message.Message, error) {
	var c, t uint16
	var ok bool

	if c, ok = dns.StringToClass[r.Class]; !ok {
		return nil, fmt.Errorf("bad class %q", r.Class)
	}
	if t, ok = dns.StringToType[r.Type]; !ok {
		return nil, fmt.Errorf("bad type %q", r.Type)
	}

	return &message.Message{
		Class:   c,
		Type:    t,
		TTL:     r.TTL,
		Content: []byte(r.Content),
	}, nil
}
