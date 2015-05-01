package backend

import (
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

type AutoBackend struct {
	Encode         yaml.MapSlice `yaml:"encode"`
	Filler         bool          `yaml:"filler"`
	Prefix, Suffix string
	SOA            string
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

func (r *AutoBackend) Check() (err error) {
	if r.Encode != nil {
		if r.encoders, err = loadEncoders(r.Encode); err != nil {
			return
		}
	}

	for zone, answer := range r.Answers {
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
			if r.encoders == nil {
				return fmt.Errorf("No encoders for zone %q and no default", zone)
			}
			log.Printf("auto: using default encoders for zone %q", zone)
			for _, e := range r.encoders {
				answer.encoders = append(answer.encoders, e)
			}
		} else {
			if answer.encoders, err = loadEncoders(answer.Encode); err != nil {
				return err
			}
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

func (r *AutoBackend) Query(m *message.Message) ([]*message.Message, error) {
	log.Printf("auto: query for %s (%s)\n", m.Name, dns.TypeToString[m.Type])

	if typesA[m.Type] {
		return r.queryA(m)
	}
	if typesAAAA[m.Type] {
		return r.queryAAAA(m)
	}
	if typesPTR[m.Type] {
		return r.queryPTR(m)
	}
	return nil, nil
}

func (r *AutoBackend) queryForward(m *message.Message, accept func(ip net.IP) bool) (rs []*message.Message, err error) {
	rs = make([]*message.Message, 0)

	for _, answer := range r.Answers {
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
		log.Printf("auto: forward %s (stripped)", name)
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
			log.Printf("auto: forward request to %s", net.IP(ip))
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
				rs = append(rs, p)
			}
			if len(ip) > 4 && ip.To16() != nil && !isCanonicalIPv4(ip) {
				p.Type = dns.TypeAAAA
				p.Content = []byte(ip.String())
				rs = append(rs, p)
			}
		}
	}

	return rs, nil
}

func (r *AutoBackend) queryA(m *message.Message) (rs []*message.Message, err error) {
	return r.queryForward(m, func(ip net.IP) bool {
		return ip.To4() != nil
	})
}

func (r *AutoBackend) queryAAAA(m *message.Message) (rs []*message.Message, err error) {
	return r.queryForward(m, func(ip net.IP) bool {
		return ip.To16() != nil
	})
}

func (r *AutoBackend) queryPTR(m *message.Message) (rs []*message.Message, err error) {
	var ip net.IP
	rs = make([]*message.Message, 0)
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

	log.Printf("auto: PTR for %s\n", ip)
	for _, answer := range r.Answers {
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
			log.Printf("auto: %T(%v) encoder created: %q", encoder, ipb.Bytes(), content)
			break
		}

		p.Content = []byte(answer.Prefix)
		p.Content = append(p.Content, []byte(content)...)
		p.Content = append(p.Content, []byte(answer.Suffix)...)
		p.Content = append(p.Content, '.')
		p.Content = append(p.Content, []byte(answer.Zone)...)
		rs = append(rs, p)
	}

	return rs, nil
}

func isCanonicalIPv4(ip net.IP) bool {
	log.Printf("canonical? %v", []byte(ip))
	if ip.To16() == nil {
		return false
	}
	for i := 0; i < len(ip); i++ {
		if ip[i] != 0 {
			return false
		}
	}
	return ip[10] == 0xff && ip[11] == 0xff
}
