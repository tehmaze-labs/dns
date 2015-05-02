package backend

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/tehmaze-labs/dns/message"
)

type Record struct {
	Class   string `yaml:"class"`
	Type    string `yaml:"type"`
	TTL     int    `yaml:"ttl"`
	Content string `yaml:"content"`
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

var defaultSOA = NewSOA()

type SOA struct {
	Source, Contact             string
	Serial                      uint64
	Refresh, Retry, Expire, TTL uint32
}

func NewSOA() *SOA {
	return &SOA{
		Source:  "localhost",
		Contact: "hostmaster.localhost",
		Serial:  1,
		Refresh: 3600,
		Retry:   600,
		Expire:  86400,
		TTL:     3600,
	}
}

func (s *SOA) Copy() *SOA {
	return &SOA{
		Source:  s.Source,
		Contact: s.Contact,
		Serial:  s.Serial,
		Refresh: s.Refresh,
		Retry:   s.Retry,
		Expire:  s.Expire,
		TTL:     s.TTL,
	}
}

func (s *SOA) Bytes() []byte {
	return []byte(s.String())
}

func (s *SOA) String() string {
	source := pickStr(s.Source, defaultSOA.Source)
	contact := strings.Replace(pickStr(s.Contact, defaultSOA.Contact), "@", ".", 1)

	return fmt.Sprintf("%s. %s. %d %d %d %d %d",
		strings.TrimRight(source, "."),
		strings.TrimRight(contact, "."),
		picku64(s.Serial, defaultSOA.Serial),
		picku32(s.Refresh, defaultSOA.Refresh),
		picku32(s.Retry, defaultSOA.Retry),
		picku32(s.Expire, defaultSOA.Expire),
		picku32(s.TTL, defaultSOA.TTL))
}
