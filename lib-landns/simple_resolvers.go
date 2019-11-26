package landns

import (
	"github.com/miekg/dns"
)

type Resolver interface {
	Resolve(Request) (Response, error)
}

type SimpleAddressResolver map[string][]AddressRecord

func (r SimpleAddressResolver) Resolve(req Request) (resp Response, err error) {
	resp.Authoritative = true

	switch req.Qtype {
	case dns.TypeA:
		for _, r := range r[req.Name] {
			if r.IsV4() {
				resp.Records = append(resp.Records, r)
			}
		}
	case dns.TypeAAAA:
		for _, r := range r[req.Name] {
			if !r.IsV4() {
				resp.Records = append(resp.Records, r)
			}
		}
	}
	return
}

type SimpleTxtResolver map[string][]TxtRecord

func (r SimpleTxtResolver) Resolve(req Request) (resp Response, err error) {
	resp.Authoritative = true

	if req.Qtype == dns.TypeTXT {
		for _, t := range r[req.Name] {
			resp.Records = append(resp.Records, t)
		}
	}
	return
}

type SimplePtrResolver map[string][]PtrRecord

func (r SimplePtrResolver) Resolve(req Request) (resp Response, err error) {
	resp.Authoritative = true

	if req.Qtype == dns.TypePTR {
		for _, p := range r[req.Name] {
			resp.Records = append(resp.Records, p)
		}
	}
	return
}

type SimpleCnameResolver map[string][]CnameRecord

func (r SimpleCnameResolver) Resolve(req Request) (resp Response, err error) {
	resp.Authoritative = true

	if req.Qtype == dns.TypeCNAME {
		for _, p := range r[req.Name] {
			resp.Records = append(resp.Records, p)
		}
	}
	return
}

type SimpleSrvResolver map[string][]SrvRecord

func (r SimpleSrvResolver) Resolve(req Request) (resp Response, err error) {
	resp.Authoritative = true

	if req.Qtype == dns.TypeSRV {
		for _, s := range r[req.Name] {
			resp.Records = append(resp.Records, s)
		}
	}
	return
}

type ResolverSet []Resolver

func (rs ResolverSet) Resolve(req Request) (resp Response, err error) {
	resp.Authoritative = true

	for _, r := range rs {
		rr, err := r.Resolve(req)
		if err != nil {
			return resp, err
		}
		resp.Records = append(resp.Records, rr.Records...)
		if !rr.Authoritative {
			resp.Authoritative = false
		}
	}
	return
}
