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

	a, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeA})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(a) != 2 {
			t.Errorf("unexcepted resolve response: %d", len(a))
		}
		if a[0].String() != "example.com. 0 A 127.1.2.3" {
			t.Errorf("unexcepted resolve response: %v", a[0].String())
		}
		if a[1].String() != "example.com. 0 A 127.2.3.4" {
			t.Errorf("unexcepted resolve response: %v", a[1].String())
		}
	}

	b, err := resolver.Resolve(dns.Question{Name: "blanktar.jp.", Qtype: dns.TypeA})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(b) != 1 {
			t.Errorf("unexcepted resolve response: %d", len(b))
		}
		if b[0].String() != "blanktar.jp. 0 A 127.2.2.2" {
			t.Errorf("unexcepted resolve response: %v", b[0].String())
		}
	}

	c, err := resolver.Resolve(dns.Question{Name: "blanktar.jp.", Qtype: dns.TypeAAAA})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(c) != 1 {
			t.Errorf("unexcepted resolve response: %d", len(c))
		}
		if c[0].String() != "blanktar.jp. 0 AAAA 4::2" {
			t.Errorf("unexcepted resolve response: %v", c[0].String())
		}
	}

	d, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeTXT})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(d) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(d))
	}

	e, err := resolver.Resolve(dns.Question{Name: "empty.example.com.", Qtype: dns.TypeA})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else if len(e) != 0 {
		t.Errorf("unexcepted resolve response: %d", len(e))
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

	a, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeTXT})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(a) != 1 {
			t.Errorf("unexcepted resolve response: %d", len(a))
		}
		if a[0].String() != `example.com. 0 TXT "hello"` {
			t.Errorf("unexcepted resolve response: %v", a[0].String())
		}
	}

	b, err := resolver.Resolve(dns.Question{Name: "blanktar.jp.", Qtype: dns.TypeTXT})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(b) != 2 {
			t.Errorf("unexcepted resolve response: %d", len(b))
		}
		if b[0].String() != `blanktar.jp. 0 TXT "foo"` {
			t.Errorf("unexcepted resolve response: %v", b[0].String())
		}
		if b[1].String() != `blanktar.jp. 0 TXT "bar"` {
			t.Errorf("unexcepted resolve response: %v", b[1].String())
		}
	}

	c, err := resolver.Resolve(dns.Question{Name: "empty.example.com.", Qtype: dns.TypeTXT})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(c) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(c))
		}
	}

	d, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeA})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(d) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(d))
		}
	}
}

func TestSimplePtrResolverResolver(t *testing.T) {
	resolver := landns.SimplePtrResolver{
		"example.com.": []landns.PtrRecord{
			landns.PtrRecord{Name: landns.Domain("example.com."), Domain: landns.Domain("target.local.")},
		},
	}

	a, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypePTR})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(a) != 1 {
			t.Errorf("unexcepted resolve response: %d", len(a))
		}
		if a[0].String() != `example.com. 0 PTR target.local.` {
			t.Errorf("unexcepted resolve response: %v", a[0].String())
		}
	}

	b, err := resolver.Resolve(dns.Question{Name: "empty.example.com.", Qtype: dns.TypePTR})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(b) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(b))
		}
	}

	c, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeTXT})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(c) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(c))
		}
	}
}

func TestSimpleCnameResolverResolver(t *testing.T) {
	resolver := landns.SimpleCnameResolver{
		"example.com.": []landns.CnameRecord{
			landns.CnameRecord{Name: landns.Domain("example.com."), Target: landns.Domain("target.local.")},
		},
	}

	a, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeCNAME})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(a) != 1 {
			t.Errorf("unexcepted resolve response: %d", len(a))
		}
		if a[0].String() != `example.com. 0 CNAME target.local.` {
			t.Errorf("unexcepted resolve response: %v", a[0].String())
		}
	}

	b, err := resolver.Resolve(dns.Question{Name: "empty.example.com.", Qtype: dns.TypePTR})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(b) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(b))
		}
	}

	c, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeTXT})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(c) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(c))
		}
	}
}

func TestSimpleSrvResolverResolver(t *testing.T) {
	resolver := landns.SimpleSrvResolver{
		"example.com.": []landns.SrvRecord{
			landns.SrvRecord{Name: landns.Domain("example.com."), Service: "http", Target: landns.Domain("target.local.")},
		},
	}

	a, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeSRV})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(a) != 1 {
			t.Errorf("unexcepted resolve response: %d", len(a))
		}
		if a[0].String() != `_http._tcp.example.com. 0 IN SRV 0 0 0 target.local.` {
			t.Errorf("unexcepted resolve response: %v", a[0].String())
		}
	}

	b, err := resolver.Resolve(dns.Question{Name: "empty.example.com.", Qtype: dns.TypeSRV})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(b) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(b))
		}
	}

	c, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeA})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(c) != 0 {
			t.Errorf("unexcepted resolve response: %d", len(c))
		}
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

	a, err := resolver.Resolve(dns.Question{Name: "example.com.", Qtype: dns.TypeA})
	if err != nil {
		t.Errorf("failed to resolve: %v", err.Error())
	} else {
		if len(a) != 2 {
			t.Errorf("unexcepted resolve response: %d", len(a))
		}
		if a[0].String() != `example.com. 0 A 127.1.2.3` {
			t.Errorf("unexcepted resolve response: %v", a[0].String())
		}
		if a[1].String() != `example.com. 0 A 127.2.3.4` {
			t.Errorf("unexcepted resolve response: %v", a[1].String())
		}
	}
}
