package encoder

import (
	"math/big"
	"net"
	"testing"
)

var key = big.NewInt(0)

func init() {
	key.SetBytes([]byte("\xc0\xca\xc0\x1a\xf0\x0d\xde\xad\xbe\xef\xba\xbe\xca\xfe\xd0\x0d"))
}

func TestBase32EncodeIPv4(t *testing.T) {
	tests := map[string]string{
		"127.0.0.1":    "fs00008",
		"172.23.42.69": "lgbikh8",
	}

	e := NewBase32()

	for test, want := range tests {
		got, err := e.Encode(net.ParseIP(test).To4())
		if err != nil {
			t.Error(err)
		} else if got != want {
			t.Errorf("got %q, want %q for %q", got, want, test)
		} else {
			t.Logf("test %q encoded to %q", test, got)
		}
	}
}

func TestBase32EncodeIPv6(t *testing.T) {
	tests := map[string]string{
		"::1": "04",
		"fe80::863a:4bff:fe11:fd1c": "vq000000000011hq9fvvs4ft3g",
	}

	e := NewBase32()

	for test, want := range tests {
		got, err := e.Encode(net.ParseIP(test).To16())
		if err != nil {
			t.Error(err)
		} else if got != want {
			t.Errorf("got %q, want %q for %q", got, want, test)
		} else {
			t.Logf("test %q encoded to %q", test, got)
		}
	}
}

func TestBase32Decode(t *testing.T) {
	tests := map[string]net.IP{
		"04": net.ParseIP("::1"),
		"vq000000000011hq9fvvs4ft3g": net.ParseIP("fe80::863a:4bff:fe11:fd1c"),
		"fs00008":                    net.ParseIP("127.0.0.1"),
	}

	e := NewBase32()

	for test, want := range tests {
		if want.To4() != nil {
			want = want.To4()
		}

		got, err := e.Decode(test)
		if err != nil {
			t.Error(err)
			continue
		}
		for len(got) < len(want) {
			got = append([]byte{0x00}, got...)
		}
		if !want.Equal(got) {
			t.Errorf("got %q (%v), want %q (%v) for %q", got, []byte(got), want, []byte(want), test)
			continue
		} else {
			t.Logf("test %q decoded to %q", test, got)
		}
	}
}
