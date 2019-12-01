package landns

import (
	"net"
	"fmt"
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

type AddressRecordConfig struct {
	TTL     *uint32 `json:"ttl,omitempty"`
	Address net.IP  `json:"address"`
}

func (a AddressRecordConfig) ToRecord(name Domain) AddressRecord {
	af := a.Normalized()
	return AddressRecord{
		Name:    name,
		TTL:     *af.TTL,
		Address: af.Address,
	}
}

func (a AddressRecordConfig) Normalized() AddressRecordConfig {
	if a.TTL == nil {
		a.TTL = &DefaultTTL
	}
	return a
}

type AddressesConfig map[Domain][]AddressRecordConfig

func (ac AddressesConfig) RegisterRecord(r AddressRecord) {
	if _, ok := ac[r.Name]; !ok {
		ac[r.Name] = []AddressRecordConfig{}
	}
	ac[r.Name] = append(ac[r.Name], AddressRecordConfig{
		TTL:     &r.TTL,
		Address: r.Address,
	})
}

func (ac AddressesConfig) Validate() error {
	for name, list := range ac {
		if err := name.Validate(); err != nil {
			return err
		}

		for _, x := range list {
			if err := x.ToRecord(name).Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

type CnameRecordConfig struct {
	TTL    *uint32 `json:"ttl,omitempty"`
	Target Domain  `json:"target"`
}

func (c CnameRecordConfig) ToRecord(name Domain) CnameRecord {
	cf := c.Normalized()
	return CnameRecord{
		Name:   name,
		TTL:    *cf.TTL,
		Target: cf.Target,
	}
}

func (c CnameRecordConfig) Normalized() CnameRecordConfig {
	if c.TTL == nil {
		c.TTL = &DefaultTTL
	}
	c.Target = c.Target.Normalized()
	return c
}

type CnamesConfig map[Domain][]CnameRecordConfig

func (cc CnamesConfig) RegisterRecord(r CnameRecord) {
	if _, ok := cc[r.Name]; !ok {
		cc[r.Name] = []CnameRecordConfig{}
	}
	cc[r.Name] = append(cc[r.Name], CnameRecordConfig{
		TTL:    &r.TTL,
		Target: r.Target,
	})
}

func (cc CnamesConfig) Validate() error {
	for name, list := range cc {
		if err := name.Validate(); err != nil {
			return err
		}

		for _, x := range list {
			if err := x.ToRecord(name).Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

type TxtRecordConfig struct {
	TTL  *uint32 `json:"ttl,omitempty"`
	Text string  `json:"text"`
}

func (t TxtRecordConfig) ToRecord(name Domain) TxtRecord {
	tf := t.Normalized()
	return TxtRecord{
		Name: name,
		TTL:  *tf.TTL,
		Text: tf.Text,
	}
}

func (t TxtRecordConfig) Normalized() TxtRecordConfig {
	if t.TTL == nil {
		t.TTL = &DefaultTTL
	}
	return t
}

type TextsConfig map[Domain][]TxtRecordConfig

func (tc TextsConfig) RegisterRecord(r TxtRecord) {
	if _, ok := tc[r.Name]; !ok {
		tc[r.Name] = []TxtRecordConfig{}
	}
	tc[r.Name] = append(tc[r.Name], TxtRecordConfig{
		TTL:  &r.TTL,
		Text: r.Text,
	})
}

func (tc TextsConfig) Validate() error {
	for name, list := range tc {
		if err := name.Validate(); err != nil {
			return err
		}

		for _, x := range list {
			if err := x.ToRecord(name).Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

type SrvRecordConfig struct {
	TTL      *uint32 `json:"ttl,omitempty"`
	Priority uint16  `json:"priority,omitempty"`
	Weight   uint16  `json:"weight,omitempty"`
	Port     uint16  `json:"port"`
	Target   Domain  `json:"target"`
}

func (s SrvRecordConfig) ToRecord(name Domain) SrvRecord {
	sf := s.Normalized()

	return SrvRecord{
		Name:     name,
		TTL:      *sf.TTL,
		Priority: sf.Priority,
		Weight:   sf.Weight,
		Port:     sf.Port,
		Target:   sf.Target,
	}
}

func (s SrvRecordConfig) Normalized() SrvRecordConfig {
	if s.TTL == nil {
		s.TTL = &DefaultTTL
	}

	s.Target = s.Target.Normalized()

	return s
}

type ServicesConfig map[Domain][]SrvRecordConfig

func (sc ServicesConfig) RegisterRecord(r SrvRecord) {
	if _, ok := sc[r.Name]; !ok {
		sc[r.Name] = []SrvRecordConfig{}
	}
	sc[r.Name] = append(sc[r.Name], SrvRecordConfig{
		TTL:      &r.TTL,
		Priority: r.Priority,
		Weight:   r.Weight,
		Port:     r.Port,
		Target:   r.Target,
	})
}

func (sc ServicesConfig) Validate() error {
	for name, list := range sc {
		if err := name.Validate(); err != nil {
			return err
		}

		for _, x := range list {
			if err := x.ToRecord(name).Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

type ResolverConfig struct {
	Addresses AddressesConfig `json:"address"`
	Cnames    CnamesConfig    `json:"cnames"`
	Texts     TextsConfig     `json:"texts"`
	Services  ServicesConfig  `json:"services"`
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
