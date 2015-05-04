package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"os"
	"path"
	"strings"

	"github.com/tehmaze-labs/dns/backend"
	"gopkg.in/yaml.v2"
)

var syslogFacility = map[string]syslog.Priority{
	"kern":   syslog.LOG_KERN,
	"user":   syslog.LOG_USER,
	"mail":   syslog.LOG_MAIL,
	"daemon": syslog.LOG_DAEMON,
	"auth":   syslog.LOG_AUTH,
	"syslog": syslog.LOG_SYSLOG,
	"lpr":    syslog.LOG_LPR,
	"news":   syslog.LOG_NEWS,
	"uucp":   syslog.LOG_UUCP,
	"cron":   syslog.LOG_CRON,
	"ftp":    syslog.LOG_FTP,
	"local0": syslog.LOG_LOCAL0,
	"local1": syslog.LOG_LOCAL1,
	"local2": syslog.LOG_LOCAL2,
	"local3": syslog.LOG_LOCAL3,
	"local4": syslog.LOG_LOCAL4,
	"local5": syslog.LOG_LOCAL5,
	"local6": syslog.LOG_LOCAL6,
	"local7": syslog.LOG_LOCAL7,
}

type Config struct {
	Backend   *backend.BackendConfig `yaml:"backend"`
	Templates interface{}            `yaml:"templates"`
	Options   struct {
		Syslog string
	}
}

func NewConfig(filename string) (c *Config, err error) {
	var data []byte

	if data, err = ioutil.ReadFile(filename); err != nil {
		return nil, err
	}

	c = &Config{}
	if err = yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}

	// Log to syslog if requested
	if c.Options.Syslog != "" {
		c.Options.Syslog = strings.ToLower(c.Options.Syslog)
		if f, found := syslogFacility[c.Options.Syslog]; found {
			l, err := syslog.New(syslog.LOG_NOTICE|f, path.Base(os.Args[0]))
			if err != nil {
				return nil, err
			}

			log.SetFlags(log.Lshortfile)
			log.SetOutput(l)
		} else {
			return nil, fmt.Errorf("Unknown syslog facility %q", c.Options.Syslog)
		}
	}

	return
}

func (c *Config) Backends() (bs []backend.Backend, err error) {
	bs = make([]backend.Backend, 0)

	for _, b := range c.Backend.AutoBackends {
		if err = b.Check(); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	for _, b := range c.Backend.GeoBackends {
		if err = b.Check(); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}

	if len(bs) == 0 {
		return nil, errors.New("no backends configured")
	}

	return
}
