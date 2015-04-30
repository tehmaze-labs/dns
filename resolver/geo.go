package resolver

import (
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
	"github.com/tehmaze-labs/dns/message"
)

var continents = []string{"AF", "AS", "EU", "NA", "OC", "SA"}

type GeoResolver struct {
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

func (r *GeoResolver) Check() (err error) {
	r.geoIP, err = geoip2.Open(r.Options.Database)
	if err != nil || r.geoIP == nil {
		return fmt.Errorf("error reading GeoIP datbases %q: %v", r.Options.Database, err)
	}

	// Normalize
	for n, zn := range r.Zones {
		r.Zones[n] = strings.ToLower(zn)
	}

	r.Options.Default.Continent = strings.ToUpper(r.Options.Default.Continent)
	r.Options.Default.Country = strings.ToUpper(r.Options.Default.Country)

	for n, rr := range r.Options.Answers.Continent {
		nu := strings.ToUpper(n)
		if n != nu {
			r.Options.Answers.Continent[nu] = rr
			delete(r.Options.Answers.Continent, n)
			n = nu
		}
	}
	for n, rr := range r.Options.Answers.Country {
		nu := strings.ToUpper(n)
		if n != nu {
			r.Options.Answers.Country[nu] = rr
			delete(r.Options.Answers.Country, n)
			n = nu
		}
	}

	return nil
}

func (r *GeoResolver) Query(m *message.Message) (a []*message.Message, err error) {
	if !stringInSlice(strings.ToLower(string(m.Name)), r.Zones) {
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

	gi, err := r.geoIP.Country(m.RemoteAddr)
	if err == nil {
		cc = gi.Country.IsoCode
		cn = gi.Country.Names["en"]
		co = gi.Continent.Code
	}

	if cc == "" {
		cc = "XX"
		cn = "Unknown"
		if r.Options.Default.Continent != "" {
			co = r.Options.Default.Continent
		} else {
			co = "EU"
		}
		if r.Options.Default.Country != "" {
			cc = r.Options.Default.Country
		} else {
			cc = "XX"
		}
	}

	a = make([]*message.Message, 0)
	if qtypes[dns.TypeTXT] {
		a = append(a, &message.Message{
			Name:    m.Name,
			Class:   dns.ClassINET,
			Type:    dns.TypeTXT,
			ID:      m.ID,
			Content: []byte(fmt.Sprintf("dns geo result for %s in %s (%s)", m.RemoteAddr, cn, co)),
		})
	}

	var records []*Record

	if answers, ok := r.Options.Answers.Continent[co]; ok {
		for _, answer := range answers {
			records = append(records, answer)
		}
	}
	if answers, ok := r.Options.Answers.Country[cc]; ok {
		for _, answer := range answers {
			records = append(records, answer)
		}
	}

	for _, record := range records {
		if !qtypes[dns.StringToType[record.Type]] {
			continue
		}

		rm, err := record.Message()
		if err != nil {
			log.Printf("bogus record: %v", err)
			continue
		}

		rm.Name = m.Name
		rm.ID = m.ID
		a = append(a, rm)
	}

	return
}

// Interface check
var _ Resolver = (*GeoResolver)(nil)
