package landns

import (
	"fmt"
	"net"
)

var (
	DefaultTTL uint32 = 3600
)

type InvalidProto string

func (p InvalidProto) Error() string {
	return fmt.Sprintf("invalid proto: \"%s\"", string(p))
}

type Proto string

func (p Proto) String() string {
	if string(p) == "" {
		return "tcp"
	}
	return string(p)
}

func (p Proto) Normalized() Proto {
	return Proto(p.String())
}

func (p Proto) Validate() error {
	if p.String() != "" && p.String() != "tcp" && p.String() != "udp" {
		return InvalidProto(p.String())
	}
	return nil
}

func (p *Proto) UnmarshalText(text []byte) error {
	if string(text) == "" {
		*p = "tcp"
	} else {
		*p = Proto(string(text))
	}

	return p.Validate()
}

func (p Proto) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

type SrvRecordShortConfig struct {
	Service  string `yaml:"service"`
	Proto    Proto  `yaml:"proto,omitempty"`
	Priority uint16 `yaml:"priority,omitempty"`
	Weight   uint16 `yaml:"weight,omitempty"`
	Port     uint16 `yaml:"port"`
	Target   Domain `yaml:"target"`
}

func (s SrvRecordShortConfig) ToRecord(name Domain, ttl uint32) SrvRecord {
	return SrvRecord{
		Name:     Domain(fmt.Sprintf("_%s._%s.%s", s.Service, s.Proto.Normalized(), name)),
		TTL:      ttl,
		Priority: s.Priority,
		Weight:   s.Weight,
		Port:     s.Port,
		Target:   s.Target,
	}
}

type ResolverShortConfig struct {
	TTL       *uint32                           `yaml:"ttl,omitempty"`
	Addresses map[Domain][]net.IP               `yaml:"address,omitempty"`
	Cnames    map[Domain][]Domain               `yaml:"cname,omitempty"`
	Texts     map[Domain][]string               `yaml:"text,omitempty"`
	Services  map[Domain][]SrvRecordShortConfig `yaml:"service,omitempty"`
}
