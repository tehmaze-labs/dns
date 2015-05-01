package encoder

import "fmt"

type Encoder interface {
	Config(opt map[string]interface{}) (err error)
	Encode(src []byte) (out string, err error)
	Decode(src string) (out []byte, err error)
}

func NewEncoder(t string, opt map[string]interface{}) (e Encoder, err error) {
	switch t {
	case "base32":
		e = NewBase32()
	case "eui64":
		e = NewEUI64()
	default:
		return nil, fmt.Errorf("No encoder with type %q found", t)
	}

	if opt != nil {
		err = e.Config(opt)
	}
	return
}
