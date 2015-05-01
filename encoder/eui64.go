package encoder

// http://standards.ieee.org/develop/regauth/tut/eui64.pdf

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const hexDigit = "0123456789abcdef"

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

type EUI64 struct {
	vendors map[string]string
}

func NewEUI64() *EUI64 {
	return &EUI64{map[string]string{}}
}

func (e *EUI64) Config(opt map[string]interface{}) (err error) {
	for k, v := range opt {
		if k != "oui" {
			return fmt.Errorf("Unknown eui64 option %q", k)
		}

		if filename, found := v.(string); found {
			e.ParseOUI(filename)
		}
	}

	return
}

func (e *EUI64) Decode(src string) (out []byte, err error) {
	p := strings.Split(src, "-")
	if len(p) < 6 {
		return nil, errors.New("No encoded OUI found")
	}
	for len(p) > 6 {
		p = p[1:]
	}

	var i, ih, il uint64

	for j := 0; j < 6; j++ {
		k, err := strconv.ParseUint(p[j], 16, 8)
		if err != nil {
			return nil, err
		}
		i = (i << 8) | k
	}

	ih = (i >> 24) ^ 0x20000
	il = (i & 0xffffff)
	out = append(out, uint8(ih>>16))
	out = append(out, uint8(ih>>8))
	out = append(out, uint8(ih))
	out = append(out, 0xff)
	out = append(out, 0xfe)
	out = append(out, uint8(il>>16))
	out = append(out, uint8(il>>8))
	out = append(out, uint8(il))
	return out, nil
}

func (e *EUI64) Encode(src []byte) (out string, err error) {
	if len(src) < 8 {
		return "", errors.New("Not a valid EUI64 address")
	}

	// Test to see if this is an EUI64 address
	a := binary.BigEndian.Uint64(src)
	if a&0xfffe000000 == 0 {
		log.Printf("eui64 encode: not an EUI64 address %032x\n", a)
		return "", errors.New("Not a valid EUI64 address")
	}

	ih := (a >> 40) ^ 0x020000
	il := a & 0x00ffffff

	buf := make([]byte, 12)
	binary.BigEndian.PutUint64(buf, (ih<<20)|il)

	return fmt.Sprintf("%s-%02x-%02x-%02x-%02x-%02x-%02x",
		e.Vendor(fmt.Sprintf("%06x", ih)),
		(ih>>16)&0xff, (ih>>8)&0xff, ih&0xff,
		(il>>16)&0xff, (il>>8)&0xff, il&0xff), nil

}

func (e *EUI64) ParseOUI(filename string) error {
	log.Printf("eui64: parsing %s\n", filename)

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
			e.vendors[fmt.Sprintf("%06x", oui)] = parseVendor(com)
		}
	}

	return nil
}

func (e *EUI64) Vendor(oui string) string {
	log.Printf("eui64: looking up vendor for %q", oui)
	var v = ""
	if len(oui) >= 3 {
		v = e.vendors[oui[:6]]
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

	return v
}

// Interface completeness validation
var _ = (*EUI64)(nil)
