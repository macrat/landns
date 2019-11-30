package landns

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

type InvalidDomain string

func (d InvalidDomain) Error() string {
	return fmt.Sprintf("invalid domain: \"%s\"", string(d))
}

type InvalidService string

func (s InvalidService) Error() string {
	return fmt.Sprintf("invalid service: \"%s\"", string(s))
}

type InvalidProto string

func (p InvalidProto) Error() string {
	return fmt.Sprintf("invalid proto: \"%s\"", string(p))
}

type InvalidPort uint16

func (p InvalidPort) Error() string {
	return fmt.Sprintf("invalid port: \"%d\"", uint16(p))
}

type Domain string

func (d Domain) String() string {
	return dns.Fqdn(string(d))
}

func (d Domain) Normalized() Domain {
	return Domain(d.String())
}

func (d Domain) Validate() error {
	if len(string(d)) == 0 {
		return InvalidDomain(string(d))
	}
	if _, ok := dns.IsDomainName(string(d)); !ok {
		return InvalidDomain(string(d))
	}
	return nil
}

func (d *Domain) UnmarshalText(text []byte) error {
	*d = Domain(dns.Fqdn(string(text)))

	return d.Validate()
}

func (d Domain) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
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

type Record interface {
	fmt.Stringer

	GetQtype() uint16
	GetName() Domain
	ToRR() (dns.RR, error)
	Validate() error
}

type TxtRecord struct {
	Name Domain
	TTL  uint16
	Text string
}

func (r TxtRecord) String() string {
	return fmt.Sprintf("%s %d TXT \"%s\"", r.Name, r.TTL, r.Text)
}

func (r TxtRecord) GetName() Domain {
	return r.Name
}

func (r TxtRecord) GetQtype() uint16 {
	return dns.TypeTXT
}

func (r TxtRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

func (r TxtRecord) Validate() error {
	return r.Name.Validate()
}

type PtrRecord struct {
	Name   Domain
	TTL    uint16
	Domain Domain
}

func (r PtrRecord) String() string {
	return fmt.Sprintf("%s %d PTR %s", r.Name, r.TTL, r.Domain)
}

func (r PtrRecord) GetName() Domain {
	return r.Name
}

func (r PtrRecord) GetQtype() uint16 {
	return dns.TypePTR
}

func (r PtrRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

func (r PtrRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Domain.Validate()
}

type CnameRecord struct {
	Name   Domain
	TTL    uint16
	Target Domain
}

func (r CnameRecord) String() string {
	return fmt.Sprintf("%s %d CNAME %s", r.Name, r.TTL, r.Target)
}

func (r CnameRecord) GetName() Domain {
	return r.Name
}

func (r CnameRecord) GetQtype() uint16 {
	return dns.TypeCNAME
}

func (r CnameRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

func (r CnameRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Target.Validate()
}

type AddressRecord struct {
	Name    Domain
	TTL     uint16
	Address net.IP
}

func (r AddressRecord) IsV4() bool {
	return r.Address.To4() != nil
}

func (r AddressRecord) String() string {
	qtype := "A"
	if !r.IsV4() {
		qtype = "AAAA"
	}
	return fmt.Sprintf("%s %d %s %s", r.Name, r.TTL, qtype, r.Address)
}

func (r AddressRecord) GetName() Domain {
	return r.Name
}

func (r AddressRecord) GetQtype() uint16 {
	if r.IsV4() {
		return dns.TypeA
	}

	return dns.TypeAAAA
}

func (r AddressRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

func (r AddressRecord) Validate() error {
	return r.Name.Validate()
}

type SrvRecord struct {
	Name     Domain
	TTL      uint16
	Service  string
	Proto    Proto
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   Domain
}

func (r SrvRecord) String() string {
	return fmt.Sprintf(
		"_%s._%s.%s %d IN SRV %d %d %d %s",
		r.Service,
		r.Proto,
		r.Name,
		r.TTL,
		r.Priority,
		r.Weight,
		r.Port,
		r.Target,
	)
}

func (r SrvRecord) GetName() Domain {
	return r.Name
}

func (r SrvRecord) GetQtype() uint16 {
	return dns.TypeSRV
}

func (r SrvRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

func (r SrvRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	if len(r.Service) <= 0 {
		return InvalidService(r.Service)
	}
	if err := r.Proto.Validate(); err != nil {
		return err
	}
	if r.Port == 0 {
		return InvalidPort(r.Port)
	}
	return r.Target.Validate()
}
