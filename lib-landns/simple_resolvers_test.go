package landns_test

import (
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func TestSimpleAddressResolver(t *testing.T) {
	resolver := landns.SimpleAddressResolver{
		"example.com.": []landns.AddressRecord{
			landns.AddressRecord{Name: landns.Domain("example.com."), Address: net.ParseIP("127.1.2.3")},
			landns.AddressRecord{Name: landns.Domain("example.com."), Address: net.ParseIP("127.2.3.4")},
		},
		"blanktar.jp.": []landns.AddressRecord{
			landns.AddressRecord{Name: landns.Domain("blanktar.jp."), Address: net.ParseIP("127.2.2.2")},
			landns.AddressRecord{Name: landns.Domain("blanktar.jp."), Address: net.ParseIP("4::2")},
		},
	}

	a, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeA, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(a.Records) != 2 {
		t.Errorf("unexcepted resolve response: %d", len(a.Records))
	} else {
		if !a.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if a.Records[0].String() != "example.com. 0 A 127.1.2.3" {
			t.Errorf("unexcepted resolve response: %v", a.Records[0].String())
		}
		if a.Records[1].String() != "example.com. 0 A 127.2.3.4" {
			t.Errorf("unexcepted resolve response: %v", a.Records[1].String())
		}
	}

	b, err := resolver.Resolve(landns.NewRequest("blanktar.jp.", dns.TypeA, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(b.Records) != 1 {
		t.Errorf("unexcepted resolve response: %d", len(b.Records))
	} else {
		if !b.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if b.Records[0].String() != "blanktar.jp. 0 A 127.2.2.2" {
			t.Errorf("unexcepted resolve response: %v", b.Records[0].String())
		}
	}

	c, err := resolver.Resolve(landns.NewRequest("blanktar.jp.", dns.TypeAAAA, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(c.Records) != 1 {
		t.Errorf("unexcepted resolve response: %d", len(c.Records))
	} else {
		if !c.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if c.Records[0].String() != "blanktar.jp. 0 AAAA 4::2" {
			t.Errorf("unexcepted resolve response: %v", c.Records[0].String())
		}
	}

	d, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeTXT, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(d.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(d.Records))
	} else if !d.Authoritative {
		t.Errorf("response was not authoritative")
	}

	e, err := resolver.Resolve(landns.NewRequest("empty.example.com.", dns.TypeA, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(e.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(e.Records))
	} else if !e.Authoritative {
		t.Errorf("response was not authoritative")
	}
}

func TestSimpleTxtResolverResolver(t *testing.T) {
	resolver := landns.SimpleTxtResolver{
		"example.com.": []landns.TxtRecord{
			landns.TxtRecord{Name: landns.Domain("example.com."), Text: "hello"},
		},
		"blanktar.jp.": []landns.TxtRecord{
			landns.TxtRecord{Name: landns.Domain("blanktar.jp."), Text: "foo"},
			landns.TxtRecord{Name: landns.Domain("blanktar.jp."), Text: "bar"},
		},
	}

	a, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeTXT, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(a.Records) != 1 {
		t.Errorf("unexcepted resolve response: %d", len(a.Records))
	} else {
		if !a.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if a.Records[0].String() != `example.com. 0 TXT "hello"` {
			t.Errorf("unexcepted resolve response: %v", a.Records[0].String())
		}
	}

	b, err := resolver.Resolve(landns.NewRequest("blanktar.jp.", dns.TypeTXT, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(b.Records) != 2 {
		t.Errorf("unexcepted resolve response: %d", len(b.Records))
	} else {
		if !b.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if b.Records[0].String() != `blanktar.jp. 0 TXT "foo"` {
			t.Errorf("unexcepted resolve response: %v", b.Records[0].String())
		}
		if b.Records[1].String() != `blanktar.jp. 0 TXT "bar"` {
			t.Errorf("unexcepted resolve response: %v", b.Records[1].String())
		}
	}

	c, err := resolver.Resolve(landns.NewRequest("empty.example.com.", dns.TypeTXT, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(c.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(c.Records))
	} else if !c.Authoritative {
		t.Errorf("response was not authoritative")
	}

	d, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeA, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(d.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(d.Records))
	} else if !d.Authoritative {
		t.Errorf("response was not authoritative")
	}
}

func TestSimplePtrResolverResolver(t *testing.T) {
	resolver := landns.SimplePtrResolver{
		"example.com.": []landns.PtrRecord{
			landns.PtrRecord{Name: landns.Domain("example.com."), Domain: landns.Domain("target.local.")},
		},
	}

	a, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypePTR, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(a.Records) != 1 {
		t.Errorf("unexcepted resolve response: %d", len(a.Records))
	} else {
		if !a.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if a.Records[0].String() != `example.com. 0 PTR target.local.` {
			t.Errorf("unexcepted resolve response: %v", a.Records[0].String())
		}
	}

	b, err := resolver.Resolve(landns.NewRequest("empty.example.com.", dns.TypePTR, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(b.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(b.Records))
	} else if !b.Authoritative {
		t.Errorf("response was not authoritative")
	}

	c, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeTXT, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(c.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(c.Records))
	} else if !c.Authoritative {
		t.Errorf("response was not authoritative")
	}
}

func TestSimpleCnameResolverResolver(t *testing.T) {
	resolver := landns.SimpleCnameResolver{
		"example.com.": []landns.CnameRecord{
			landns.CnameRecord{Name: landns.Domain("example.com."), Target: landns.Domain("target.local.")},
		},
	}

	a, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeCNAME, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(a.Records) != 1 {
		t.Errorf("unexcepted resolve response: %d", len(a.Records))
	} else {
		if !a.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if a.Records[0].String() != `example.com. 0 CNAME target.local.` {
			t.Errorf("unexcepted resolve response: %v", a.Records[0].String())
		}
	}

	b, err := resolver.Resolve(landns.NewRequest("empty.example.com.", dns.TypePTR, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(b.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(b.Records))
	} else if !b.Authoritative {
		t.Errorf("response was not authoritative")
	}

	c, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeTXT, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(c.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(c.Records))
	} else if !c.Authoritative {
		t.Errorf("response was not authoritative")
	}
}

func TestSimpleSrvResolverResolver(t *testing.T) {
	resolver := landns.SimpleSrvResolver{
		"example.com.": []landns.SrvRecord{
			landns.SrvRecord{Name: landns.Domain("example.com."), Service: "http", Target: landns.Domain("target.local.")},
		},
	}

	a, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeSRV, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(a.Records) != 1 {
		t.Errorf("unexcepted resolve response: %d", len(a.Records))
	} else {
		if !a.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if a.Records[0].String() != `_http._tcp.example.com. 0 IN SRV 0 0 0 target.local.` {
			t.Errorf("unexcepted resolve response: %v", a.Records[0].String())
		}
	}

	b, err := resolver.Resolve(landns.NewRequest("empty.example.com.", dns.TypeSRV, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(b.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(b.Records))
	} else if !b.Authoritative {
		t.Errorf("response was not authoritative")
	}

	c, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeA, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(c.Records) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(c.Records))
	} else if !c.Authoritative {
		t.Errorf("response was not authoritative")
	}
}

func TestResolverSet(t *testing.T) {
	resolverA := landns.SimpleAddressResolver{
		"example.com.": []landns.AddressRecord{
			landns.AddressRecord{Name: landns.Domain("example.com."), Address: net.ParseIP("127.1.2.3")},
		},
	}
	resolverB := landns.SimpleAddressResolver{
		"example.com.": []landns.AddressRecord{
			landns.AddressRecord{Name: landns.Domain("example.com."), Address: net.ParseIP("127.2.3.4")},
		},
	}
	resolver := landns.ResolverSet{resolverA, resolverB}

	a, err := resolver.Resolve(landns.NewRequest("example.com.", dns.TypeA, false))
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(a.Records) != 2 {
		t.Errorf("unexcepted resolve response: %d", len(a.Records))
	} else {
		if !a.Authoritative {
			t.Errorf("response was not authoritative")
		}
		if a.Records[0].String() != `example.com. 0 A 127.1.2.3` {
			t.Errorf("unexcepted resolve response: %v", a.Records[0].String())
		}
		if a.Records[1].String() != `example.com. 0 A 127.2.3.4` {
			t.Errorf("unexcepted resolve response: %v", a.Records[1].String())
		}
	}
}

type DummyAuthoritativeResolver bool

func (d DummyAuthoritativeResolver) Resolve(r landns.Request) (landns.Response, error) {
	return landns.Response{Authoritative: bool(d)}, nil
}

func TestResolverSet_Authoritative(t *testing.T) {
	resolverT := DummyAuthoritativeResolver(true)
	resolverF := DummyAuthoritativeResolver(false)

	req := landns.NewRequest("example.com.", dns.TypeA, false)

	tests := []struct {
		Name          string
		Resolver      landns.Resolver
		Authoritative bool
	}{
		{"true-true", landns.ResolverSet{resolverT, resolverT}, true},
		{"true-false", landns.ResolverSet{resolverT, resolverF}, false},
		{"false-true", landns.ResolverSet{resolverF, resolverT}, false},
		{"false-false", landns.ResolverSet{resolverF, resolverT}, false},
	}

	for _, test := range tests {
		if resp, err := test.Resolver.Resolve(req); err != nil {
			t.Errorf("failed to resolve: %s: %v", test.Name, err.Error())
		} else {
			if len(resp.Records) != 0 {
				t.Errorf("unexcepted resolve response: %s: %d", test.Name, len(resp.Records))
			}
			if resp.Authoritative != test.Authoritative {
				t.Errorf("unexcepted authoritative: %s: excepted:%v != got:%v", test.Name, test.Authoritative, resp.Authoritative)
			}
		}
	}
}
