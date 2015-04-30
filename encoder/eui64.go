package encoder

// http://standards.ieee.org/develop/regauth/tut/eui64.pdf

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var vendorStrip = regexp.MustCompile(`[^- a-z0-9]`)
var vendorDashes = regexp.MustCompile(`-[-]*`)
var vendorStopWords = map[string]bool{
	"bv":            true,
	"company":       true,
	"co":            true,
	"communication": true,
	"corp":          true,
	"corporate":     true,
	"corporation":   true,
	"coltd":         true,
	"devices":       true,
	"electronica":   true,
	"electronics":   true,
	"gmbh":          true,
	"inc":           true,
	"int":           true,
	"international": true,
	"limited":       true,
	"llg":           true,
	"ltd":           true,
	"manufacturing": true,
	"srl":           true,
	"systemes":      true,
	"systems":       true,
	"technologies":  true,
	"technology":    true,
	"the":           true,
}

type EUI64Encoder struct {
	fallback Encoder
	vendors  map[string]string
}

func NewEUI64(fallback Encoder) *EUI64Encoder {
	return &EUI64Encoder{fallback, map[string]string{}}
}

func (e *EUI64Encoder) Encode(ip net.IP) string {
	if _, _, err := net.ParseCIDR(ip.String() + "/64"); err != nil {
		return ""
	}

	n := binary.BigEndian.Uint64(ip[8:])

	// Test to see if this is an EUI64 address
	if n&0xfffe000000 == 0 {
		if e.fallback != nil {
			return e.fallback.Encode(ip)
		}
		return ""
	}

	ih := (n >> 40) ^ 0x020000
	il := n & 0x00ffffff

	return fmt.Sprintf("%02x-%02x-%02x-%02x-%02x-%02x",
		(ih>>16)&0xff, (ih>>8)&0xff, ih&0xff,
		(il>>16)&0xff, (il>>8)&0xff, il&0xff)
}

func (e *EUI64Encoder) Decode(s string) (ip net.IP, err error) {
	p := strings.SplitN(s, "-", 6)
	if len(p) != 6 {
		if e.fallback != nil {
			return e.fallback.Decode(s)
		}
		return nil, errors.New("No encoded OUI found")
	}

	var i, ih, il uint64

	for j := 0; j < 6; j++ {
		k, err := strconv.ParseUint(p[j], 16, 8)
		if err != nil {
			if e.fallback != nil {
				return e.fallback.Decode(s)
			}
			return nil, err
		}
		i = (i << 8) | k
	}

	ih = (i >> 24) ^ 0x20000
	il = (i & 0xffffff)
	ip = make([]byte, 16)
	ip[0x08] = uint8(ih >> 16)
	ip[0x09] = uint8(ih >> 8)
	ip[0x0a] = uint8(ih)
	ip[0x0b] = 0xff
	ip[0x0c] = 0xfe
	ip[0x0d] = uint8(il >> 16)
	ip[0x0e] = uint8(il >> 8)
	ip[0x0f] = uint8(il)
	return ip, nil
}

func (e *EUI64Encoder) ParseOUI(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	s := bufio.NewScanner(r)

	// 00-00-00   (hex)              XEROX CORPORATION
	for s.Scan() {
		line := s.Text()
		oui := make([]byte, 3)

		if _, err := fmt.Sscanf(line, "  %02x-%02x-%02x   (hex)\t\t", &oui[0], &oui[1], &oui[2]); err == nil {
			com := (strings.Split(line, "\t"))[2]
			e.vendors[string(oui)] = parseVendor(com)
			fmt.Printf("%q -> %q\n", com, parseVendor(com))
		}
	}

	return nil
}

func (e *EUI64Encoder) Vendor(oui string) string {
	var v = ""
	if len(oui) >= 3 {
		v = e.vendors[oui[:3]]
	}
	if v == "" {
		return "unknown"
	}
	return v
}

func parseVendor(v string) string {
	var o, p []string

	v = strings.ToLower(v)
	v = strings.SplitN(v, "&", 2)[0]
	v = vendorStrip.ReplaceAllString(v, "")
	p = strings.Split(v, " ")
	for _, s := range p {
		if vendorStopWords[s] {
			continue
		}
		o = append(o, s)
	}
	v = strings.Join(o, "-")
	v = vendorDashes.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")

	// Hostnames can't start with a number
	/*
		if len(v) > 1 && v[0] >= '0' && v[0] <= '9' {
			v = "x" + v
		}
	*/

	return v
}

// Interface completeness validation
var _ Encoder = (*EUI64Encoder)(nil)
