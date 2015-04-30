package encoder

import (
	"fmt"
	"net"
)

type Encoder interface {
	Encode(ip net.IP) string
	Decode(string) (net.IP, error)
}

func NewEncoder(t string) (Encoder, error) {
	switch t {
	case "base32":
		return NewBase32(), nil
	case "eui64":
		return NewEUI64(nil), nil
	case "eui64+base32":
		return NewEUI64(NewBase32()), nil
	}

	return nil, fmt.Errorf("No encoder with type %q found", t)
}
