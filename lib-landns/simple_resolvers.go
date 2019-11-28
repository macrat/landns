package landns

import (
	"fmt"

	"github.com/miekg/dns"
)

type DomainIsNotFQDNError string

func (e DomainIsNotFQDNError) Error() string {
	return fmt.Sprintf(`error: "%s" domain is not FQDN`, string(e))
}

type Resolver interface {
	Resolve(ResponseWriter, Request) error
}

type ValidatableResolver interface {
	Resolver

	Validate() error
}

type SimpleAddressResolver map[string][]AddressRecord

func (r SimpleAddressResolver) Resolve(resp ResponseWriter, req Request) error {
	switch req.Qtype {
	case dns.TypeA:
		for _, a := range r[req.Name] {
			if a.IsV4() {
				if err := resp.Add(a); err != nil {
					return err
				}
			}
		}
	case dns.TypeAAAA:
		for _, a := range r[req.Name] {
			if !a.IsV4() {
				if err := resp.Add(a); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r SimpleAddressResolver) Validate() error {
	for name, records := range r {
		if !dns.IsFqdn(name) {
			return DomainIsNotFQDNError(name)
		}
		for _, record := range records {
			if err := record.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimpleAddressResolver) String() string {
	records := 0
	for _, rs := range r {
		records += len(rs)
	}
	return fmt.Sprintf("SimpleAddressResolver[%d domains %d records]", len(r), records)
}

type SimpleTxtResolver map[string][]TxtRecord

func (r SimpleTxtResolver) Resolve(resp ResponseWriter, req Request) error {
	if req.Qtype == dns.TypeTXT {
		for _, t := range r[req.Name] {
			if err := resp.Add(t); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimpleTxtResolver) Validate() error {
	for name, records := range r {
		if !dns.IsFqdn(name) {
			return DomainIsNotFQDNError(name)
		}
		for _, record := range records {
			if err := record.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimpleTxtResolver) String() string {
	records := 0
	for _, rs := range r {
		records += len(rs)
	}
	return fmt.Sprintf("SimpleTxtResolver[%d domains %d records]", len(r), records)
}

type SimplePtrResolver map[string][]PtrRecord

func (r SimplePtrResolver) Resolve(resp ResponseWriter, req Request) error {
	if req.Qtype == dns.TypePTR {
		for _, p := range r[req.Name] {
			if err := resp.Add(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimplePtrResolver) Validate() error {
	for name, records := range r {
		if !dns.IsFqdn(name) {
			return DomainIsNotFQDNError(name)
		}
		for _, record := range records {
			if err := record.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimplePtrResolver) String() string {
	records := 0
	for _, rs := range r {
		records += len(rs)
	}
	return fmt.Sprintf("SimplePtrResolver[%d domains %d records]", len(r), records)
}

type SimpleCnameResolver map[string][]CnameRecord

func (r SimpleCnameResolver) Resolve(resp ResponseWriter, req Request) error {
	if req.Qtype == dns.TypeCNAME {
		for _, p := range r[req.Name] {
			if err := resp.Add(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimpleCnameResolver) Validate() error {
	for name, records := range r {
		if !dns.IsFqdn(name) {
			return DomainIsNotFQDNError(name)
		}
		for _, record := range records {
			if err := record.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimpleCnameResolver) String() string {
	records := 0
	for _, rs := range r {
		records += len(rs)
	}
	return fmt.Sprintf("SimpleCnameResolver[%d domains %d records]", len(r), records)
}

type SimpleSrvResolver map[string][]SrvRecord

func (r SimpleSrvResolver) Resolve(resp ResponseWriter, req Request) error {
	if req.Qtype == dns.TypeSRV {
		for _, s := range r[req.Name] {
			if err := resp.Add(s); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimpleSrvResolver) Validate() error {
	for name, records := range r {
		if !dns.IsFqdn(name) {
			return DomainIsNotFQDNError(name)
		}
		for _, record := range records {
			if err := record.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r SimpleSrvResolver) String() string {
	records := 0
	for _, rs := range r {
		records += len(rs)
	}
	return fmt.Sprintf("SimpleSrvResolver[%d domains %d records]", len(r), records)
}

type ResolverSet []Resolver

func (rs ResolverSet) Resolve(resp ResponseWriter, req Request) error {
	for _, r := range rs {
		if err := r.Resolve(resp, req); err != nil {
			return err
		}
	}
	return nil
}

func (rs ResolverSet) String() string {
	return fmt.Sprintf("ResolverSet%s", []Resolver(rs))
}
