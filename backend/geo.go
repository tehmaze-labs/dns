package backend

import (
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
	"github.com/tehmaze-labs/dns/message"
)

var continents = []string{"AF", "AS", "EU", "NA", "OC", "SA"}

type GeoBackend struct {
	Zones   []string `yaml:"zones"`
	Options struct {
		Database string `yaml:"database"`
		Answers  struct {
			Continent map[string][]*Record
			Country   map[string][]*Record
		}
		Default struct {
			Continent string
			Country   string
		}
	}

	geoIP *geoip2.Reader
}

func (b *GeoBackend) Check() (err error) {
	b.geoIP, err = geoip2.Open(b.Options.Database)
	if err != nil || b.geoIP == nil {
		return fmt.Errorf("error reading GeoIP datbases %q: %v", b.Options.Database, err)
	}

	// Normalize
	for n, zn := range b.Zones {
		b.Zones[n] = strings.ToLower(zn)
	}

	b.Options.Default.Continent = strings.ToUpper(b.Options.Default.Continent)
	b.Options.Default.Country = strings.ToUpper(b.Options.Default.Country)

	if err = b.checkAnswers(b.Options.Answers.Continent); err != nil {
		return
	}
	if err = b.checkAnswers(b.Options.Answers.Country); err != nil {
		return
	}

	return nil
}

func (r *GeoBackend) checkAnswers(answers map[string][]*Record) (err error) {
	if answers == nil {
		return
	}
	for n, a := range answers {
		nu := strings.ToUpper(n)
		if n != nu {
			answers[nu] = a
			delete(answers, n)
			n = nu
		}

		// Basic record pre-flight checks
		for _, r := range a {
			if r.Class == "" {
				r.Class = dns.ClassToString[dns.ClassINET]
			}
			if _, ok := dns.StringToClass[r.Class]; !ok {
				return fmt.Errorf("Unknown class %q", r.Class)
			}
			if _, ok := dns.StringToType[r.Type]; !ok {
				return fmt.Errorf("Unknown type %q", r.Type)
			}
		}
	}
	return
}

func (b *GeoBackend) Query(m *message.Message) (r []*message.Message, err error) {
	if !stringInSlice(strings.ToLower(string(m.Name)), b.Zones) {
		return nil, nil
	}

	qtypes := map[uint16]bool{
		dns.TypeAAAA: true,
		dns.TypeA:    true,
		dns.TypeTXT:  true,
	}
	if m.Type != dns.TypeANY {
		qtypes = map[uint16]bool{
			m.Type: true,
		}
	}

	var cc, cn, co string

	gi, err := b.geoIP.Country(m.RemoteAddr)
	if err == nil {
		cc = gi.Country.IsoCode
		cn = gi.Country.Names["en"]
		co = gi.Continent.Code
	}

	if cc == "" {
		cc = "XX"
		cn = "Unknown"
		if b.Options.Default.Continent != "" {
			co = b.Options.Default.Continent
		} else {
			co = "EU"
		}
		if b.Options.Default.Country != "" {
			cc = b.Options.Default.Country
		} else {
			cc = "XX"
		}
	}

	r = make([]*message.Message, 0)
	if qtypes[dns.TypeTXT] {
		r = append(r, &message.Message{
			Name:    m.Name,
			Class:   dns.ClassINET,
			Type:    dns.TypeTXT,
			ID:      m.ID,
			Content: []byte(fmt.Sprintf("dns geo result for %s in %s (%s)", m.RemoteAddr, cn, co)),
		})
	}

	var records []*Record

	if answers, ok := b.Options.Answers.Continent[co]; ok {
		for _, answer := range answers {
			records = append(records, answer)
		}
	}
	if answers, ok := b.Options.Answers.Country[cc]; ok {
		for _, answer := range answers {
			records = append(records, answer)
		}
	}

	for _, record := range records {
		if !qtypes[dns.StringToType[record.Type]] {
			continue
		}

		p, err := record.Message()
		if err != nil {
			log.Printf("bogus record: %v", err)
			continue
		}

		p.Name = m.Name
		p.ID = m.ID
		r = append(r, p)
	}

	return
}

// Interface check
var _ Backend = (*GeoBackend)(nil)
