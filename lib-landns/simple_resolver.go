package landns

import (
	"net"

	"github.com/go-yaml/yaml"
	"github.com/miekg/dns"
)

type SimpleResolver map[uint16]map[Domain][]Record

func NewSimpleResolver(records []Record) SimpleResolver {
	sr := make(SimpleResolver)

	for _, r := range records {
		qtype := r.GetQtype()
		name := r.GetName()

		if _, ok := sr[qtype]; !ok {
			sr[qtype] = make(map[Domain][]Record)
		}
		sr[qtype][name] = append(sr[qtype][name], r)
	}

	return sr
}

func (sr SimpleResolver) Resolve(w ResponseWriter, r Request) error {
	domains := sr[r.Qtype]
	if domains == nil {
		return nil
	}
	for _, record := range domains[Domain(r.Name)] {
		w.Add(record)
	}
	return nil
}

func (sr SimpleResolver) Validate() error {
	for _, domains := range sr {
		for _, records := range domains {
			for _, r := range records {
				if err := r.Validate(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func makeReverseMap(addresses map[Domain][]net.IP, ttl uint16) ([]Record, error) {
	reverse := []Record{}

	for addr, ips := range addresses {
		for _, ip := range ips {
			key, err := dns.ReverseAddr(ip.String())
			if err != nil {
				return nil, err
			}

			reverse = append(reverse, PtrRecord{
				Name:   Domain(key),
				TTL:    ttl,
				Domain: addr,
			})
		}
	}

	return reverse, nil
}

func NewSimpleResolverFromConfig(config []byte) (SimpleResolver, error) {
	var conf ResolverShortConfig
	if err := yaml.Unmarshal(config, &conf); err != nil {
		return SimpleResolver{}, err
	}

	ttl := uint16(3600)
	if conf.TTL != nil {
		ttl = *conf.TTL
	}

	records := []Record{}

	for addr, ips := range conf.Addresses {
		for _, ip := range ips {
			records = append(records, AddressRecord{
				Name:    addr,
				TTL:     ttl,
				Address: ip,
			})
		}
	}

	reverse, err := makeReverseMap(conf.Addresses, ttl)
	if err != nil {
		return SimpleResolver{}, err
	}
	records = append(records, reverse...)

	for addr, targets := range conf.Cnames {
		for _, t := range targets {
			records = append(records, CnameRecord{
				Name:   addr,
				TTL:    ttl,
				Target: t,
			})
		}
	}

	for addr, texts := range conf.Texts {
		for _, t := range texts {
			records = append(records, TxtRecord{
				Name: addr,
				TTL:  ttl,
				Text: t,
			})
		}
	}

	for addr, services := range conf.Services {
		for _, s := range services {
			srv := s.ToRecord(addr, ttl)
			if err := srv.Validate(); err != nil {
				return SimpleResolver{}, err
			}
			records = append(records, srv)
		}
	}

	return NewSimpleResolver(records), nil
}
