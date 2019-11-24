package landns

import (
	"github.com/miekg/dns"
)

type Resolver interface {
	Resolve(dns.Question) ([]Record, error)
}

type SimpleAddressResolver map[string][]AddressRecord

func (r SimpleAddressResolver) Resolve(q dns.Question) (resp []Record, err error) {
	switch q.Qtype {
	case dns.TypeA:
		for _, r := range r[q.Name] {
			if r.IsV4() {
				resp = append(resp, r)
			}
		}
	case dns.TypeAAAA:
		for _, r := range r[q.Name] {
			if !r.IsV4() {
				resp = append(resp, r)
			}
		}
	}
	return
}

type SimpleTxtResolver map[string][]TxtRecord

func (r SimpleTxtResolver) Resolve(q dns.Question) (resp []Record, err error) {
	if q.Qtype == dns.TypeTXT {
		for _, t := range r[q.Name] {
			resp = append(resp, t)
		}
	}
	return
}

type SimplePtrResolver map[string][]PtrRecord

func (r SimplePtrResolver) Resolve(q dns.Question) (resp []Record, err error) {
	if q.Qtype == dns.TypePTR {
		for _, p := range r[q.Name] {
			resp = append(resp, p)
		}
	}
	return
}

type SimpleCnameResolver map[string][]CnameRecord

func (r SimpleCnameResolver) Resolve(q dns.Question) (resp []Record, err error) {
	if q.Qtype == dns.TypeCNAME {
		for _, p := range r[q.Name] {
			resp = append(resp, p)
		}
	}
	return
}

type SimpleSrvResolver map[string][]SrvRecord

func (r SimpleSrvResolver) Resolve(q dns.Question) (resp []Record, err error) {
	if q.Qtype == dns.TypeSRV {
		for _, s := range r[q.Name] {
			resp = append(resp, s)
		}
	}
	return
}

type ResolverSet []Resolver

func (rs ResolverSet) Resolve(q dns.Question) (resp []Record, err error) {
	for _, r := range rs {
		rr, err := r.Resolve(q)
		if err != nil {
			return nil, err
		}
		resp = append(resp, rr...)
	}
	return
}
