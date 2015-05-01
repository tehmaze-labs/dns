package encoder

import (
	"encoding/base32"
	"math/big"
	"strings"
)

var base32pads = map[byte]bool{
	'=':  true,
	0x00: true,
}

type Base32 struct {
	shift *big.Int
	xor   *big.Int
}

func NewBase32() *Base32 {
	return &Base32{}
}

func (e *Base32) Config(opt map[string]interface{}) (err error) {
	return
}

func (e *Base32) Encode(src []byte) (out string, err error) {
	// Trim left zero bytes
	for len(src) > 0 && src[0] == 0x00 {
		src = src[1:]
	}

	out = base32.HexEncoding.EncodeToString(src)
	out = strings.TrimRight(out, "\x00=")
	out = strings.ToLower(out)
	return
}

func (e *Base32) Decode(src string) (out []byte, err error) {
	// Upper case only
	tmp := strings.ToUpper(src)

	// Realign padding
	for len(tmp)%8 > 0 {
		tmp += "="
	}

	return base32.HexEncoding.DecodeString(tmp)
}

// Interface completeness validation
var _ = (*Base32)(nil)
