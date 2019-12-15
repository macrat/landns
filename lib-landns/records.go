package landns

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

var (
	ErrUnsupportedType = fmt.Errorf("unsupported record type")
	ErrInvalidRecord   = fmt.Errorf("invalid format record")
)

// InvalidDomain is error for invalid domain.
type InvalidDomain string

// Error is getter for description string.
func (d InvalidDomain) Error() string {
	return fmt.Sprintf("invalid domain: \"%s\"", string(d))
}

// InvalidPort is error for invalid port.
type InvalidPort uint16

// Error is getter for description string.
func (p InvalidPort) Error() string {
	return fmt.Sprintf("invalid port: \"%d\"", uint16(p))
}

// Domain is the type for domain.
type Domain string

// String is converter to Domain to string.
//
// String will convert to FQDN.
func (d Domain) String() string {
	return dns.Fqdn(string(d))
}

// Normalized is normalizer for make FQDN.
func (d Domain) Normalized() Domain {
	return Domain(d.String())
}

// Validate is validator to domain.
func (d Domain) Validate() error {
	if len(string(d)) == 0 {
		return InvalidDomain(string(d))
	}
	if _, ok := dns.IsDomainName(string(d)); !ok {
		return InvalidDomain(string(d))
	}
	return nil
}

// UnmarshalText is parse text from bytes.
func (d *Domain) UnmarshalText(text []byte) error {
	*d = Domain(dns.Fqdn(string(text)))

	return d.Validate()
}

// MarshalText is make bytes text.
func (d Domain) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// Record is the record entry of DNS.
type Record interface {
	fmt.Stringer

	GetQtype() uint16      // Get query type like dns.TypeA of package github.com/miekg/dns.
	GetName() Domain       // Get domain name of the Record.
	GetTTL() uint32        // Get TTL for the Record.
	ToRR() (dns.RR, error) // Get response record for dns.Msg of package github.com/miekg/dns.
	Validate() error       // Validation Record and returns error if invalid.
}

// NewRecord is make new Record from query string.
func NewRecord(str string) (Record, error) {
	rr, err := dns.NewRR(str)
	if err != nil {
		return nil, err
	}

	return NewRecordFromRR(rr)
}

// NewRecordFromRR is make new Record from dns.RR of package github.com/miekg/dns.
func NewRecordFromRR(rr dns.RR) (Record, error) {
	switch x := rr.(type) {
	case *dns.A:
		return AddressRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Address: x.A}, nil
	case *dns.AAAA:
		return AddressRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Address: x.AAAA}, nil
	case *dns.CNAME:
		return CnameRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Target: Domain(x.Target)}, nil
	case *dns.PTR:
		return PtrRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Domain: Domain(x.Ptr)}, nil
	case *dns.SRV:
		return SrvRecord{
			Name:     Domain(x.Hdr.Name),
			TTL:      x.Hdr.Ttl,
			Priority: x.Priority,
			Weight:   x.Weight,
			Port:     x.Port,
			Target:   Domain(x.Target),
		}, nil
	case *dns.TXT:
		return TxtRecord{Name: Domain(x.Hdr.Name), TTL: x.Hdr.Ttl, Text: x.Txt[0]}, nil
	default:
		return nil, ErrUnsupportedType
	}
}

// AddressRecord is the Record of A or AAAA.
type AddressRecord struct {
	Name    Domain
	TTL     uint32
	Address net.IP
}

// IsV4 is checker for guess that which of IPv4 (A record) or IPv6 (AAAA record).
func (r AddressRecord) IsV4() bool {
	return r.Address.To4() != nil
}

// String is make record string.
func (r AddressRecord) String() string {
	qtype := "A"
	if !r.IsV4() {
		qtype = "AAAA"
	}
	return fmt.Sprintf("%s %d IN %s %s", r.Name, r.TTL, qtype, r.Address)
}

// GetName is getter to name of record.
func (r AddressRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r AddressRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r AddressRecord) GetQtype() uint16 {
	if r.IsV4() {
		return dns.TypeA
	}

	return dns.TypeAAAA
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r AddressRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

// Validate is validator of record.
func (r AddressRecord) Validate() error {
	return r.Name.Validate()
}

// CnameRecord is the Record of CNAME.
type CnameRecord struct {
	Name   Domain
	TTL    uint32
	Target Domain
}

// String is make record string.
func (r CnameRecord) String() string {
	return fmt.Sprintf("%s %d IN CNAME %s", r.Name, r.TTL, r.Target)
}

// GetName is getter to name of record.
func (r CnameRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r CnameRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r CnameRecord) GetQtype() uint16 {
	return dns.TypeCNAME
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r CnameRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

// Validate is validator of record.
func (r CnameRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Target.Validate()
}

// PtrRecord is the Record of PTR.
type PtrRecord struct {
	Name   Domain
	TTL    uint32
	Domain Domain
}

// String is make record string.
func (r PtrRecord) String() string {
	return fmt.Sprintf("%s %d IN PTR %s", r.Name, r.TTL, r.Domain)
}

// GetName is getter to name of record.
func (r PtrRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r PtrRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r PtrRecord) GetQtype() uint16 {
	return dns.TypePTR
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r PtrRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

// Validate is validator of record.
func (r PtrRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	return r.Domain.Validate()
}

// TxtRecord is the Record of TXT.
type TxtRecord struct {
	Name Domain
	TTL  uint32
	Text string
}

// String is make record string.
func (r TxtRecord) String() string {
	return fmt.Sprintf("%s %d IN TXT \"%s\"", r.Name, r.TTL, r.Text)
}

// GetName is getter to name of record.
func (r TxtRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r TxtRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r TxtRecord) GetQtype() uint16 {
	return dns.TypeTXT
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r TxtRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

// Validate is validator of record.
func (r TxtRecord) Validate() error {
	return r.Name.Validate()
}

// SrvRecord is the Record of SRV.
type SrvRecord struct {
	Name     Domain
	TTL      uint32
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   Domain
}

// String is make record string.
func (r SrvRecord) String() string {
	return fmt.Sprintf(
		"%s %d IN SRV %d %d %d %s",
		r.Name,
		r.TTL,
		r.Priority,
		r.Weight,
		r.Port,
		r.Target,
	)
}

// GetName is getter to name of record.
func (r SrvRecord) GetName() Domain {
	return r.Name
}

// GetTTL is getter to TTL of record.
func (r SrvRecord) GetTTL() uint32 {
	return r.TTL
}

// GetQtype is getter to query type number like dns.TypeA or dns.TypeTXT of package github.com/miekg/dns.
func (r SrvRecord) GetQtype() uint16 {
	return dns.TypeSRV
}

// ToRR is converter to dns.RR of package github.com/miekg/dns
func (r SrvRecord) ToRR() (dns.RR, error) {
	return dns.NewRR(r.String())
}

// Validate is validator of record.
func (r SrvRecord) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return err
	}
	if r.Port == 0 {
		return InvalidPort(r.Port)
	}
	return r.Target.Validate()
}
