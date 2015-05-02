package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"github.com/tehmaze-labs/dns/backend"
	"github.com/tehmaze-labs/dns/message"
)

var (
	HELLO_ABI_VERSION_2 = []byte("HELO\t2")
	HELLO_REPLY         = "OK\tdns-pdns\n"
	END_REPLY           = "END\n"
	FAIL_REPLY          = "FAIL\n"
	NL                  = []byte("\n")
)

const (
	RTYPE_AXRF = "AXFR"
	RTYPE_Q    = "Q"
	RTYPE_PING = "PING"
)

type Pdns struct {
	backends []backend.Backend
}

type pdnsRequest struct {
	rtype   string
	message *message.Message
}

func New(backends []backend.Backend) *Pdns {
	return &Pdns{backends}
}

func parseRequest(line []byte) (*pdnsRequest, error) {
	tokens := bytes.Split(line, []byte("\t"))
	kind := string(tokens[0])
	switch kind {
	case RTYPE_Q:
		if len(tokens) < 7 {
			return nil, errors.New("bad request line")
		}

		var c, t uint16
		var ok bool

		if c, ok = dns.StringToClass[string(tokens[2])]; !ok {
			return nil, errors.New("bad query class")
		}
		if t, ok = dns.StringToType[string(tokens[3])]; !ok {
			return nil, errors.New("bad query type")
		}

		return &pdnsRequest{
			kind,
			&message.Message{
				Name:       tokens[1],
				Class:      c,
				Type:       t,
				ID:         tokens[4],
				RemoteAddr: net.ParseIP(string(tokens[5])),
				LocalAddr:  net.ParseIP(string(tokens[6])),
			},
		}, nil
	case RTYPE_AXRF, RTYPE_PING:
		return &pdnsRequest{
			kind,
			nil,
		}, nil
	default:
		return nil, errors.New("bad request line")
	}
}

func write(w io.Writer, line string) {
	//fmt.Fprintf(os.Stderr, ">>> %q\n", line)
	_, err := io.WriteString(w, line)
	if err != nil {
		log.Printf("write failed: %v", err)
	}
}

func (p *Pdns) Serve(r io.Reader, w io.Writer) {
	log.Print("starting mazenet-pdns backend")
	buf := bufio.NewReader(r)
	handshake := true

	for {
		line, isPrefix, err := buf.ReadLine()
		if err == nil && isPrefix {
			err = errors.New("pdns line too long")
		}
		if err == io.EOF {
			log.Print("terminating mazenet-pdns backend")
			return
		}
		if err != nil {
			write(w, fmt.Sprintf("LOG failed reading request: %v\n", err))
			//log.Printf("failed reading request: %v", err)
			continue
		}

		//fmt.Fprintf(os.Stderr, "<<< %s\n", line)

		if handshake {
			if !bytes.Equal(line, HELLO_ABI_VERSION_2) {
				log.Printf("handshake failed: %v != %v", line, HELLO_ABI_VERSION_2)
				write(w, FAIL_REPLY)
			} else {
				handshake = false
				write(w, HELLO_REPLY)
			}
			continue
		}

		req, err := parseRequest(line)
		if err != nil {
			log.Printf("failed parsing request: %v", err)
			write(w, FAIL_REPLY)
			continue
		}

		switch req.rtype {
		case RTYPE_Q:
			answers, err := p.handleRequest(req)
			if err != nil {
				log.Printf("failed handling request: %v", err)
				continue
			}

			for _, answer := range answers {
				if answer == nil {
					continue
				}
				m, err := p.marshal(answer)
				if err != nil {
					log.Printf("failed to marshal answer: %v", err)
					continue
				}
				write(w, m)
			}
			write(w, END_REPLY)
		}
	}

}

func (p *Pdns) handleRequest(req *pdnsRequest) ([]*message.Message, error) {
	if req.message == nil {
		return nil, errors.New("no dns request message")
	}

	messages := make([]*message.Message, 0)
	for _, backend := range p.backends {
		answers, err := backend.Query(req.message)
		if err != nil {
			log.Printf("backend returned error: %v", err)
			continue
		}
		messages = append(messages, answers...)
	}

	return messages, nil
}

func (p *Pdns) marshal(message *message.Message) (string, error) {
	var c, t string
	var ok bool

	if c, ok = dns.ClassToString[message.Class]; !ok {
		return "", fmt.Errorf("bad query class %d", message.Class)
	}
	if t, ok = dns.TypeToString[message.Type]; !ok {
		return "", fmt.Errorf("bad query type %d", message.Type)
	}

	m := []string{"DATA"}
	m = append(m, string(message.Name))
	m = append(m, c)
	m = append(m, t)
	m = append(m, strconv.Itoa(message.TTL))
	m = append(m, string(message.ID))
	m = append(m, string(message.Content))

	return strings.Join(m, "\t") + "\n", nil
}
