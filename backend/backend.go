package backend

import "github.com/tehmaze-labs/dns/message"

type Backend interface {
	Check() error
	Query(*message.Message) ([]*message.Message, error)
}

type BackendConfig struct {
	AutoBackends []*AutoBackend `yaml:"auto"`
	GeoBackends  []*GeoBackend  `yaml:"geo"`
}
