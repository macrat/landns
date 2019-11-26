package landns_test

import (
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func ResolverTest(t *testing.T, resolver landns.Resolver, request landns.Request, authoritative bool, responses ...string) {
	resp, err := resolver.Resolve(request)
	if err != nil {
		t.Errorf("%s <- %s: failed to resolve: %v", resolver, request, err.Error())
		return
	}

	if resp.Authoritative != authoritative {
		t.Errorf(`%s <- %s: unexcepted authoritive of response: excepted %v but got %v`, resolver, request, authoritative, resp.Authoritative)
	}

	if len(resp.Records) != len(responses) {
		t.Errorf(`%s <- %s: unexcepted resolve response: excepted length %d but got %d`, resolver, request, len(responses), len(resp.Records))
		return
	}

	l := len(resp.Records)
	if l > len(responses) {
		l = len(responses)
	}

	for i := 0; i < l; i++ {
		if resp.Records[i].String() != responses[i] {
			t.Errorf(`%s <- %s: unexcepted resolve response: excepted "%s" but got "%s"`, resolver, request, responses[i], resp.Records[i])
		}
	}
}

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

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 0 A 127.1.2.3", "example.com. 0 A 127.2.3.4")

	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeA, false), true, "blanktar.jp. 0 A 127.2.2.2")
	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeAAAA, false), true, "blanktar.jp. 0 AAAA 4::2")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeA, false), true)
	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeAAAA, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true)
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

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true, `example.com. 0 TXT "hello"`)
	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeTXT, false), true, `blanktar.jp. 0 TXT "foo"`, `blanktar.jp. 0 TXT "bar"`)

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeTXT, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true)
}

func TestSimplePtrResolverResolver(t *testing.T) {
	resolver := landns.SimplePtrResolver{
		"example.com.": []landns.PtrRecord{
			landns.PtrRecord{Name: landns.Domain("example.com."), Domain: landns.Domain("target.local.")},
		},
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypePTR, false), true, "example.com. 0 PTR target.local.")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypePTR, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true)
}

func TestSimpleCnameResolverResolver(t *testing.T) {
	resolver := landns.SimpleCnameResolver{
		"example.com.": []landns.CnameRecord{
			landns.CnameRecord{Name: landns.Domain("example.com."), Target: landns.Domain("target.local.")},
		},
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeCNAME, false), true, "example.com. 0 CNAME target.local.")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeCNAME, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true)
}

func TestSimpleSrvResolverResolver(t *testing.T) {
	resolver := landns.SimpleSrvResolver{
		"example.com.": []landns.SrvRecord{
			landns.SrvRecord{Name: landns.Domain("example.com."), Service: "http", Target: landns.Domain("target.local.")},
		},
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeSRV, false), true, "_http._tcp.example.com. 0 IN SRV 0 0 0 target.local.")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeSRV, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true)
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

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 0 A 127.1.2.3", "example.com. 0 A 127.2.3.4")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeA, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true)
}

type DummyAuthoritativeResolver bool

func (d DummyAuthoritativeResolver) Resolve(r landns.Request) (landns.Response, error) {
	return landns.Response{Authoritative: bool(d)}, nil
}

func TestResolverSet_Authoritative(t *testing.T) {
	resolverT := DummyAuthoritativeResolver(true)
	resolverF := DummyAuthoritativeResolver(false)

	req := landns.NewRequest("example.com.", dns.TypeA, false)

	ResolverTest(t, resolverT, req, true)
	ResolverTest(t, resolverF, req, false)

	ResolverTest(t, landns.ResolverSet{resolverT, resolverT}, req, true)
	ResolverTest(t, landns.ResolverSet{resolverT, resolverF}, req, false)
	ResolverTest(t, landns.ResolverSet{resolverF, resolverT}, req, false)
	ResolverTest(t, landns.ResolverSet{resolverF, resolverF}, req, false)
}
