package backend

import (
	"net"
	"testing"
)

func TestReverseNetwork(t *testing.T) {
	var tests = map[string]string{
		"1.2.3.4/5":          "in-addr.arpa",
		"127.0.0.0/8":        "127.in-addr.arpa",
		"192.168.0.0/16":     "168.192.in-addr.arpa",
		"172.16.0.0/12":      "172.in-addr.arpa",
		"2001::/3":           "ip6.arpa",
		"2001:470:d510::/48": "0.1.5.d.0.7.4.0.1.0.0.2.ip6.arpa",
		"fe80::/64":          "0.0.0.0.0.0.0.0.0.0.0.0.0.8.e.f.ip6.arpa",
		"::1/128":            "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
	}

	for test, want := range tests {
		_, ipnet, err := net.ParseCIDR(test)
		if err != nil {
			t.Fatal(err)
		}
		got := ReverseNetwork(ipnet)
		if got != want {
			t.Fatalf("got %q, want %q for %q", got, want, test)
		} else {
			t.Logf("test %q resolved to %q", test, got)
		}
	}
}
