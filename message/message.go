package message

import "net"

type Message struct {
	Name                  []byte
	Class                 uint16
	Type                  uint16
	TTL                   int
	ID                    []byte
	Content               []byte
	RemoteAddr, LocalAddr net.IP
}
