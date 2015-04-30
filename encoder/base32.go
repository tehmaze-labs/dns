package encoder

import (
	"encoding/base32"
	"math/big"
	"net"
	"strings"
)

type Base32Encoder struct {
	shift *big.Int
	xor   *big.Int
}

func NewBase32() *Base32Encoder {
	return &Base32Encoder{}
}

func NewBase32Xor(xor *big.Int) *Base32Encoder {
	return &Base32Encoder{xor: xor}
}

func NewBase32Shift(shift *big.Int) *Base32Encoder {
	return &Base32Encoder{shift: shift}
}

func (e *Base32Encoder) Encode(ip net.IP) string {
	n := pton(ip)
	if n == nil {
		return ""
	}

	if e.shift != nil {
		n = n.Add(n, e.shift)
	}
	if e.xor != nil {
		n = n.Xor(n, e.xor)
	}

	o := make([]byte, 32)
	base32.HexEncoding.Encode(o, n.Bytes())
	return strings.ToLower(strings.TrimRight(string(o), "\x00="))
}

func (e *Base32Encoder) Decode(s string) (ip net.IP, err error) {
	// Upper case only
	s = strings.ToUpper(s)

	// Realign padding
	for len(s)%8 > 0 {
		s += "="
	}
	b, err := base32.HexEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	n := new(big.Int)
	n.SetBytes(b)

	if e.xor != nil {
		n = n.Xor(n, e.xor)
	}

	ip = append(ip, n.Bytes()...)
	for len(ip) > 4 && len(ip) < 16 {
		ip = append([]byte{0x00}, ip...)
	}
	for len(ip) < 4 {
		ip = append([]byte{0x00}, ip...)
	}

	return
}

// Interface completeness validation
var _ Encoder = (*Base32Encoder)(nil)
