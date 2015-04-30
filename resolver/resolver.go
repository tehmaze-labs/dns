package resolver

import "github.com/tehmaze-labs/dns/message"

type Resolver interface {
	Check() error
	Query(*message.Message) ([]*message.Message, error)
}

type ResolverConfig struct {
	AutoResolvers []*AutoResolver `yaml:"auto"`
	GeoResolvers  []*GeoResolver  `yaml:"geo"`
}
