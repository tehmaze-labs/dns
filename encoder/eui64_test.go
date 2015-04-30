package encoder

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestEUI64Encode(t *testing.T) {
	tests := map[string]string{
		"fe80::5074:f2ff:feb1:a87f": "52-74-f2-b1-a8-7f",
		"fe80::216:3eff:fe83:f111":  "00-16-3e-83-f1-11",
		"::1": "",
	}

	e := NewEUI64Encoder(nil)
	for test, want := range tests {
		got := e.Encode(net.ParseIP(test))
		if got != want {
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

	e := NewEUI64Encoder(nil)
	for test, want := range tests {
		got, err := e.Decode(test)
		if want == nil {
			if err == nil {
				t.Error("got %q, want error", got)
			} else {
				t.Logf("test %q returned error %v", test, err)
			}
		} else {
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
		"\x00\x00\x3b": "i-controls",
		"\x00\x00\x3c": "auspex",
		"\x00\x16\x3e": "xensource",
		"\x00\x23\x42": "coffee",
		"\xff\xff\xff": "unknown",
	}

	e := NewEUI64Encoder(nil)
	e.ParseOUI("./testdata/oui.txt")

	for test, want := range tests {
		got := e.Vendor(test)
		if got != want {
			t.Errorf("got %q, want %q for %q", got, want, test)
		} else {
			t.Logf("test %q decoded to %q", formatOUI(test), got)
		}
	}
}

func formatOUI(oui string) string {
	var o = []string{}
	for i := 0; i < len(oui); i++ {
		o = append(o, fmt.Sprintf("%02x", byte(oui[i])))
	}
	return strings.Join(o, ":")
}
