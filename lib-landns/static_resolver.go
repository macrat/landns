package landns

import (
	"net"

	"github.com/go-yaml/yaml"
	"github.com/miekg/dns"
)

func makeReverseMap(addresses map[Domain][]net.IP, ttl uint16) (map[string][]PtrRecord, error) {
	reverse := map[string][]PtrRecord{}
	for addr, ips := range addresses {
		for _, ip := range ips {
			key, err := dns.ReverseAddr(ip.String())
			if err != nil {
				return nil, err
			}

			if _, ok := reverse[key]; !ok {
				reverse[key] = []PtrRecord{}
			}

			reverse[key] = append(reverse[key], PtrRecord{
				Name:   Domain(key),
				TTL:    ttl,
				Domain: addr,
			})
		}
	}
	return reverse, nil
}

type StaticResolver ResolverSet

func NewStaticResolver(config []byte) (StaticResolver, error) {
	var conf ResolverShortConfig
	if err := yaml.Unmarshal(config, &conf); err != nil {
		return StaticResolver{}, err
	}

	ttl := uint16(3600)
	if conf.TTL != nil {
		ttl = *conf.TTL
	}

	address := map[string][]AddressRecord{}
	for addr, ips := range conf.Addresses {
		address[addr.String()] = []AddressRecord{}
		for _, ip := range ips {
			address[addr.String()] = append(address[addr.String()], AddressRecord{
				Name:    addr,
				TTL:     ttl,
				Address: ip,
			})
		}
	}

	reverse, err := makeReverseMap(conf.Addresses, ttl)
	if err != nil {
		return StaticResolver{}, err
	}

	cname := map[string][]CnameRecord{}
	for addr, records := range conf.Cnames {
		cname[addr.String()] = []CnameRecord{}
		for _, t := range records {
			cname[addr.String()] = append(cname[addr.String()], CnameRecord{
				Name:   addr,
				TTL:    ttl,
				Target: t,
			})
		}
	}

	text := map[string][]TxtRecord{}
	for addr, records := range conf.Texts {
		text[addr.String()] = []TxtRecord{}
		for _, t := range records {
			text[addr.String()] = append(text[addr.String()], TxtRecord{
				Name: addr,
				TTL:  ttl,
				Text: t,
			})
		}
	}

	service := map[string][]SrvRecord{}
	for addr, records := range conf.Services {
		ss := []SrvRecord{}
		for _, x := range records {
			s := x.ToRecord(addr, ttl)
			if err := s.Validate(); err != nil {
				return StaticResolver{}, err
			}
			ss = append(ss, s)
		}
		service[addr.String()] = ss
	}

	return StaticResolver{
		SimpleAddressResolver(address),
		SimpleCnameResolver(cname),
		SimplePtrResolver(reverse),
		SimpleTxtResolver(text),
		SimpleSrvResolver(service),
	}, nil
}

func (r StaticResolver) Resolve(resp ResponseWriter, req Request) error {
	return ResolverSet(r).Resolve(resp, req)
}
