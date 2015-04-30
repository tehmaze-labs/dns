package encoder

import (
	"math/big"
	"net"
)

func pton(ip net.IP) (n *big.Int) {
	if ip.IsUnspecified() {
		return
	}
	if ip.To4() != nil {
		n = big.NewInt(0)
		n.SetBytes(ip.To4())
	} else {
		n = big.NewInt(0)
		n.SetBytes(ip)
	}
	return
}
