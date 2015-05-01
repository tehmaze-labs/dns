package encoder

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestEUI64Encode(t *testing.T) {
	tests := map[string]string{
		"fe80::216:3eff:fe83:f111":  "00-16-3e-83-f1-11",
		"fe80::5074:f2ff:feb1:a87f": "52-74-f2-b1-a8-7f",
		"fe80::608b:ccff:fe6b:82a9": "62-8b-cc-6b-82-a9",
		"::1": "",
	}

	e := NewEUI64()
	e.ParseOUI("./testdata/oui.txt")

	for test, want := range tests {
		got, err := e.Encode(net.ParseIP(test))
		if want == "" {
			if err == nil {
				t.Errorf("got %q, want error", got)
			} else {
				t.Logf("test %q returned error %v (expected)", test, err)
			}
		} else if err != nil {
			t.Error(err)
		} else if !strings.HasSuffix(got, want) {
			t.Errorf("got %q, want %q for %q", got, want, test)
		} else {
			t.Logf("test %q encoded to %q", test, got)
		}
	}
}

func TestEUI64Decode(t *testing.T) {
	tests := map[string]net.IP{
		"00-16-3e-83-f1-11": net.ParseIP("::216:3eff:fe83:f111"),
		"52-74-f2-b1-a8-7f": net.ParseIP("::5074:f2ff:feb1:a87f"),
		"":                  nil,
	}

	e := NewEUI64()
	for test, want := range tests {
		data, err := e.Decode(test)
		if want == nil {
			if err == nil {
				t.Error("got %q, want error", data)
			} else {
				t.Logf("test %q returned error %v (expected)", test, err)
			}
		} else {
			got := net.IP(data)
			for len(got) < 16 {
				got = append([]byte{0x00}, got...)
			}
			if err != nil {
				t.Error(err)
			} else if !bytes.Equal(got, want) {
				t.Errorf("got %q, want %q for %q", got, want, test)
			} else {
				t.Logf("test %q decoded to %q", test, got)
			}
		}
	}
}

func TestEUI64Vendors(t *testing.T) {
	tests := map[string]string{
		"00003b": "i-controls",
		"00003c": "auspex",
		"00163e": "xensource",
		"002342": "coffee-equipment",
		"ffffff": "unknown",
	}

	e := NewEUI64()
	e.ParseOUI("./testdata/oui.txt")

	for test, want := range tests {
		got := e.Vendor(test)
		if got != want {
			t.Errorf("got %q, want %q for %q", got, want, test)
		} else {
			t.Logf("test %q decoded to %q", test, got)
		}
	}
}
