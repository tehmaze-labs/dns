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

func TestBase32Encode(t *testing.T) {
	tests := map[string]string{
		"::1": "04",
		"fe80::863a:4bff:fe11:fd1c": "vq000000000011hq9fvvs4ft3g",
		"127.0.0.1":                 "fs00008",
	}

	e := NewBase32()

	for test, want := range tests {
		ip := net.ParseIP(test)
		got := e.Encode(ip)

		if got != want {
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

func TestBaseShift32Encode(t *testing.T) {
	tests := map[string]string{
		"fe80::863a:4bff:fe11:fd1c/64": "got4nvvu27uho",
		"127.0.0.1/8":                  "04",
	}

	for test, want := range tests {
		ip, ipnet, err := net.ParseCIDR(test)
		if err != nil {
			t.Error(err)
			continue
		}

		i := pton(ipnet.IP)
		e := NewBase32Shift(i.Mul(i, big.NewInt(-1)))
		got := e.Encode(ip)
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		} else {
			t.Logf("test %q encoded to %q", test, got)
		}
	}
}

func TestBase32XorEncode(t *testing.T) {
	tests := map[string]string{
		"fe80::863a:4bff:fe11:fd1c": "7p5c06ng1nfaqe6lu50j9rpd24",
		"127.0.0.1":                 "o35c06ng1nfarfnfnavbbvmg1g",
	}

	e := NewBase32Xor(key)
	for test, want := range tests {
		got := e.Encode(net.ParseIP(test))
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		} else {
			t.Logf("test %q encoded to %q", test, got)
		}
	}
}

func TestBase32XorDecode(t *testing.T) {
	tests := map[string]net.IP{
		"7p5c06ng1nfaqe6lu50j9rpd24": net.ParseIP("fe80::863a:4bff:fe11:fd1c"),
		"o35c06ng1nfarfnfnavbbvmg1g": net.ParseIP("127.0.0.1"),
	}

	e := NewBase32Xor(key)
	for test, want := range tests {
		got, err := e.Decode(test)
		if err != nil {
			t.Error(err)
		} else if !want.Equal(got) {
			t.Errorf("expected %q, got %q", want, got)
		} else {
			t.Logf("test %q decoded to %q", test, got)
		}
	}
}
