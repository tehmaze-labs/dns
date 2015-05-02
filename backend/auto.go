package backend

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"sort"
	"strings"

	"github.com/miekg/dns"
	"github.com/tehmaze-labs/dns/encoder"
	"github.com/tehmaze-labs/dns/message"
	"gopkg.in/yaml.v2"
)

const SOATemplate = "%s. hostmaster.localhost. 1 28800 7200 604800 86400"

var (
	typesA = map[uint16]bool{
		dns.TypeANY: true,
		dns.TypeA:   true,
	}
	typesAAAA = map[uint16]bool{
		dns.TypeANY:  true,
		dns.TypeAAAA: true,
	}
	typesNS = map[uint16]bool{
		dns.TypeANY: true,
		dns.TypeNS:  true,
	}
	typesPTR = map[uint16]bool{
		dns.TypeANY: true,
		dns.TypePTR: true,
	}
	typesSOA = map[uint16]bool{
		dns.TypeANY: true,
		dns.TypeSOA: true,
	}
)

type AutoBackend struct {
	Encode         yaml.MapSlice `yaml:"encode"`
	Filler         bool          `yaml:"filler"`
	Prefix, Suffix string
	SOA            *SOA
	DNS            []string
	Answers        map[string]*AutoBackendAnswer

	encoders []encoder.Encoder
}

type AutoBackendAnswer struct {
	Network        *net.IPNet
	Size           int
	Zone           string
	Encode         yaml.MapSlice `yaml:"encode"`
	Filler         bool          `yaml:"filler"`
	Prefix, Suffix string
	SOA            *SOA
	DNS            []string
	Version        uint8

	encoders []encoder.Encoder
	network  *big.Int
}

func loadEncoders(e yaml.MapSlice) (encoders []encoder.Encoder, err error) {
	encoders = make([]encoder.Encoder, 0)

	for _, item := range e {
		if encoderName, found := item.Key.(string); found {
			opt := map[string]interface{}{}
			if values, found := item.Value.(yaml.MapSlice); found {
				for _, value := range values {
					if k, found := value.Key.(string); found {
						opt[k] = value.Value
					} else {
						return nil, fmt.Errorf("Unknown key type %T in %v", value.Key, value)
					}
				}
			}

			encoder, err := encoder.NewEncoder(encoderName, opt)
			if err != nil {
				return nil, err
			}
			if encoder == nil {
				return nil, fmt.Errorf("Unknown error loading encoder %q", encoderName)
			}
			encoders = append(encoders, encoder)
		}
	}
	return
}

func (b *AutoBackend) Check() (err error) {
	log.Println("auto: check")
	if b.DNS == nil || len(b.DNS) == 0 {
		return errors.New("auto: no DNS servers configured")
	}
	if b.SOA == nil {
		b.SOA = NewSOA()
		b.SOA.Source = b.DNS[0]
	}
	log.Printf("auto: SOA %q\n", b.SOA.String())
	if b.Encode != nil {
		if b.encoders, err = loadEncoders(b.Encode); err != nil {
			return
		}
	}

	for zone, answer := range b.Answers {
		_, answer.Network, err = net.ParseCIDR(zone)
		if err != nil {
			return err
		}
		answer.network = new(big.Int)
		answer.network.SetBytes(answer.Network.IP)
		if answer.Size == 0 {
			ones, _ := answer.Network.Mask.Size()
			answer.Size = ones
		}
		if answer.Encode == nil {
			if b.encoders == nil {
				return fmt.Errorf("No encoders for zone %q and no default", zone)
			}
			log.Printf("auto: using default encoders for zone %q", zone)
			for _, e := range b.encoders {
				answer.encoders = append(answer.encoders, e)
			}
		} else {
			if answer.encoders, err = loadEncoders(answer.Encode); err != nil {
				return err
			}
		}
		if answer.Zone == "" {
			return fmt.Errorf("No forward zone for zone %q", zone)
		}
		if answer.Prefix == "" && b.Prefix != "" {
			answer.Prefix = b.Prefix
		}
		if answer.Suffix == "" && b.Suffix != "" {
			answer.Suffix = b.Suffix
		}
		if answer.DNS == nil {
			answer.DNS = []string{}
		}
		if len(answer.DNS) == 0 {
			for _, dns := range b.DNS {
				answer.DNS = append(answer.DNS, dns)
			}
		}
		if answer.SOA == nil {
			answer.SOA = b.SOA.Copy()
			answer.SOA.Source = answer.DNS[0]
		}
		log.Printf("auto: %s SOA %q\n", answer.Zone, answer.SOA.String())
	}

	return
}

func (b *AutoBackend) Query(m *message.Message) (r []*message.Message, err error) {
	log.Printf("auto: query for %s (%s)\n", m.Name, dns.TypeToString[m.Type])

	r = make([]*message.Message, 0)

	var replies []*message.Message
	if typesA[m.Type] {
		if replies, err = b.queryA(m); err != nil {
			return
		}
		r = b.merge(r, replies)
	}
	if typesAAAA[m.Type] {
		if replies, err = b.queryAAAA(m); err != nil {
			return
		}
		r = b.merge(r, replies)
	}
	if typesNS[m.Type] {
		if replies, err = b.queryNS(m); err != nil {
			return
		}
		r = b.merge(r, replies)
	}
	if typesPTR[m.Type] {
		if replies, err = b.queryPTR(m); err != nil {
			return
		}
		r = b.merge(r, replies)
	}
	if typesSOA[m.Type] {
		if replies, err = b.querySOA(m); err != nil {
			return
		}
		r = b.merge(r, replies)
	}

	return
}

func (b *AutoBackend) merge(r, replies []*message.Message) []*message.Message {
	if replies != nil {
		r = append(r, replies...)
	}
	return r
}

func (b *AutoBackend) queryForward(m *message.Message, accept func(ip net.IP) bool) (r []*message.Message, err error) {
	r = make([]*message.Message, 0)

	for _, answer := range b.Answers {
		name := string(m.Name)
		if !strings.HasSuffix(name, "."+answer.Zone) {
			continue
		}
		name = name[:len(name)-len(answer.Zone)-1]
		if answer.Prefix != "" && !strings.HasPrefix(name, answer.Prefix) {
			continue
		}
		name = strings.TrimPrefix(name, answer.Prefix)
		if answer.Suffix != "" && !strings.HasSuffix(name, answer.Suffix) {
			continue
		}
		name = strings.TrimSuffix(name, answer.Suffix)
		for _, encoder := range answer.encoders {
			d, err := encoder.Decode(name)
			if err != nil {
				continue
			}
			ipn := new(big.Int)
			ipn.SetBytes(d)
			ipn = ipn.Add(ipn, answer.network)
			ip := net.IP(ipn.Bytes())
			if !accept(ip) {
				continue
			}
			p := &message.Message{
				Name:  m.Name,
				Class: dns.ClassINET,
				Type:  dns.TypePTR,
				TTL:   60,
				ID:    m.ID,
			}
			if ip.To4() != nil {
				p.Type = dns.TypeA
				p.Content = []byte(ip.String())
				r = append(r, p)
			}
			if len(ip) > 4 && ip.To16() != nil && !isCanonicalIPv4(ip) {
				p.Type = dns.TypeAAAA
				p.Content = []byte(ip.String())
				r = append(r, p)
			}
		}
	}

	return
}

func (b *AutoBackend) queryA(m *message.Message) (rs []*message.Message, err error) {
	return b.queryForward(m, func(ip net.IP) bool {
		return ip.To4() != nil
	})
}

func (b *AutoBackend) queryAAAA(m *message.Message) (rs []*message.Message, err error) {
	return b.queryForward(m, func(ip net.IP) bool {
		return ip.To16() != nil && !isCanonicalIPv4(ip)
	})
}

func (b *AutoBackend) queryNS(m *message.Message) (r []*message.Message, err error) {
	r = make([]*message.Message, 0)

	var name = string(m.Name)
	for _, answer := range b.Answers {
		zone := ReverseNetwork(answer.Network)
		if zone != name {
			continue
		}

		for _, d := range answer.DNS {
			p := &message.Message{
				Name:    m.Name,
				Class:   dns.ClassINET,
				Type:    dns.TypeNS,
				TTL:     int(answer.SOA.TTL),
				ID:      m.ID,
				Content: []byte(d),
			}
			r = append(r, p)
		}
		break
	}

	return r, nil
}

func (b *AutoBackend) queryPTR(m *message.Message) (r []*message.Message, err error) {
	var ip net.IP
	name := string(m.Name)

	if strings.HasSuffix(name, ".ip6.arpa") {
		ips := stringRev(strings.Replace(name[:len(name)-8], ".", "", -1))
		if len(ips) != 32 {
			return nil, nil
		}
		ipn := new(big.Int)
		ipn.SetString(ips, 16)
		ip = net.IP(ipn.Bytes())
	} else if strings.HasSuffix(name, ".in-addr.arpa") {
		ips := sort.StringSlice(strings.Split(name[:len(name)-13], "."))
		if len(ips) != 4 {
			return nil, nil
		}
		for i, j := 0, len(ips)-1; i < j; i, j = i+1, j-1 {
			ips[i], ips[j] = ips[j], ips[i]
		}
		ip = net.ParseIP(strings.Join(ips[:], ".")).To4()
	}

	if ip == nil || ip.IsUnspecified() {
		return nil, nil
	}

	r = make([]*message.Message, 0)
	for _, answer := range b.Answers {
		if answer.Network == nil || !answer.Network.Contains(ip) {
			continue
		}
		p := &message.Message{
			Name:  m.Name,
			Class: dns.ClassINET,
			Type:  dns.TypePTR,
			TTL:   60,
			ID:    m.ID,
		}

		ipb := new(big.Int)
		ipb.SetBytes(ip)
		ipb = ipb.Xor(ipb, answer.network)

		content := ""
		for _, encoder := range answer.encoders {
			content, err = encoder.Encode(ipb.Bytes())
			if content == "" || err != nil {
				continue
			}
			break
		}

		p.Content = []byte(answer.Prefix)
		p.Content = append(p.Content, []byte(content)...)
		p.Content = append(p.Content, []byte(answer.Suffix)...)
		p.Content = append(p.Content, '.')
		p.Content = append(p.Content, []byte(answer.Zone)...)
		r = append(r, p)
	}

	return
}

func (b *AutoBackend) querySOA(m *message.Message) (r []*message.Message, err error) {
	r = make([]*message.Message, 0)

	var name = string(m.Name)
	for _, answer := range b.Answers {
		zone := ReverseNetwork(answer.Network)
		if zone != name {
			continue
		}

		p := &message.Message{
			Name:    m.Name,
			Class:   dns.ClassINET,
			Type:    dns.TypeSOA,
			TTL:     int(answer.SOA.TTL),
			ID:      m.ID,
			Content: answer.SOA.Bytes(),
		}
		r = append(r, p)
		break
	}

	return r, nil
}

func isCanonicalIPv4(ip net.IP) bool {
	if ip.To16() == nil {
		return false
	}
	if len(ip) == 4 {
		return false
	}
	for i := 0; i < len(ip); i++ {
		if ip[i] != 0 {
			return false
		}
	}
	return ip[10] == 0xff && ip[11] == 0xff
}

func ReverseNetwork(net *net.IPNet) string {
	size, _ := net.Mask.Size()
	if isCanonicalIPv4(net.IP) || net.IP.To4() != nil {
		switch {
		case size == 32:
			return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa", net.IP[3], net.IP[2], net.IP[1], net.IP[0])
		case size >= 24:
			return fmt.Sprintf("%d.%d.%d.in-addr.arpa", net.IP[2], net.IP[1], net.IP[0])
		case size >= 16:
			return fmt.Sprintf("%d.%d.in-addr.arpa", net.IP[1], net.IP[0])
		case size >= 8:
			return fmt.Sprintf("%d.in-addr.arpa", net.IP[0])
		default:
			return "in-addr.arpa"
		}
	} else {
		ip := net.IP.To16()
		hex := []byte{}
		for i := len(ip) - 1; i >= 0; i-- {
			v := ip[i]
			hex = append(hex, hexDigit[v&0xf])
			hex = append(hex, '.')
			hex = append(hex, hexDigit[v>>4])
			hex = append(hex, '.')
		}

		off := 32 - int(size/4)
		return string(hex[off*2:]) + "ip6.arpa"
	}
}
