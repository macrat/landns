package landns_test

import (
	"fmt"
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

	for i, _ := range responses {
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

	if err := resolver.Validate(); err != nil {
		t.Fatalf("failed to validate resolver: %s", err)
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 0 A 127.1.2.3", "example.com. 0 A 127.2.3.4")

	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeA, false), true, "blanktar.jp. 0 A 127.2.2.2")
	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeAAAA, false), true, "blanktar.jp. 0 AAAA 4::2")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeA, false), true)
	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeAAAA, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true)
}

func BenchmarkSimpleAddressResolver(b *testing.B) {
	resolver := landns.SimpleAddressResolver{}

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)
		resolver[host] = []landns.AddressRecord{
			{Name: landns.Domain(host), Address: net.ParseIP("127.1.2.3")},
			{Name: landns.Domain(host), Address: net.ParseIP("127.2.3.4")},
			{Name: landns.Domain(host), Address: net.ParseIP("1:2:3::4")},
			{Name: landns.Domain(host), Address: net.ParseIP("5:6:7::8")},
		}
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(req)
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

	if err := resolver.Validate(); err != nil {
		t.Fatalf("failed to validate resolver: %s", err)
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true, `example.com. 0 TXT "hello"`)
	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeTXT, false), true, `blanktar.jp. 0 TXT "foo"`, `blanktar.jp. 0 TXT "bar"`)

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeTXT, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true)
}

func BenchmarkSimpleTxtResolver(b *testing.B) {
	resolver := landns.SimpleTxtResolver{}

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)
		resolver[host] = []landns.TxtRecord{
			{Name: landns.Domain(host), Text: "hello world"},
			{Name: landns.Domain(host), Text: "this_is_test"},
		}
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeTXT, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(req)
	}
}

func TestSimplePtrResolverResolver(t *testing.T) {
	resolver := landns.SimplePtrResolver{
		"3.2.1.127.in-addr.arpa.": []landns.PtrRecord{
			landns.PtrRecord{Name: landns.Domain("3.2.1.127.in-addr.arpa."), Domain: landns.Domain("target.local.")},
		},
		"8.7.6.5.4.3.2.1.f.e.d.c.b.a.0.9.8.7.6.5.4.3.2.1.ip6.arpa.": []landns.PtrRecord{
			landns.PtrRecord{Name: landns.Domain("8.7.6.5.4.3.2.1.f.e.d.c.b.a.0.9.8.7.6.5.4.3.2.1.ip6.arpa."), Domain: landns.Domain("target.local.")},
		},
	}

	if err := resolver.Validate(); err != nil {
		t.Fatalf("failed to validate resolver: %s", err)
	}

	ResolverTest(t, resolver, landns.NewRequest("3.2.1.127.in-addr.arpa.", dns.TypePTR, false), true, "3.2.1.127.in-addr.arpa. 0 PTR target.local.")
	ResolverTest(t, resolver, landns.NewRequest("8.7.6.5.4.3.2.1.f.e.d.c.b.a.0.9.8.7.6.5.4.3.2.1.ip6.arpa.", dns.TypePTR, false), true, "8.7.6.5.4.3.2.1.f.e.d.c.b.a.0.9.8.7.6.5.4.3.2.1.ip6.arpa. 0 PTR target.local.")

	ResolverTest(t, resolver, landns.NewRequest("4.2.1.127.in-addr.arpa.", dns.TypePTR, false), true)

	ResolverTest(t, resolver, landns.NewRequest("3.2.1.127.in-addr.arpa.", dns.TypeA, false), true)
}

func BenchmarkSimplePtrResolver(b *testing.B) {
	resolver := landns.SimplePtrResolver{}

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("%d.1.168.192.in-addr.arpa.", i)
		resolver[host] = []landns.PtrRecord{
			{Name: landns.Domain(host), Domain: landns.Domain("example.com.")},
			{Name: landns.Domain(host), Domain: landns.Domain("example.com.")},
		}
	}

	req := landns.NewRequest("50.1.168.192.in-addr.arpa.", dns.TypePTR, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(req)
	}
}

func TestSimpleCnameResolverResolver(t *testing.T) {
	resolver := landns.SimpleCnameResolver{
		"example.com.": []landns.CnameRecord{
			landns.CnameRecord{Name: landns.Domain("example.com."), Target: landns.Domain("target.local.")},
		},
	}

	if err := resolver.Validate(); err != nil {
		t.Fatalf("failed to validate resolver: %s", err)
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeCNAME, false), true, "example.com. 0 CNAME target.local.")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeCNAME, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true)
}

func BenchmarkSimpleCnameResolver(b *testing.B) {
	resolver := landns.SimpleCnameResolver{}

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)
		resolver[host] = []landns.CnameRecord{
			{Name: landns.Domain(host), Target: landns.Domain("example.com.")},
			{Name: landns.Domain(host), Target: landns.Domain("example.com.")},
		}
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeCNAME, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(req)
	}
}

func TestSimpleSrvResolverResolver(t *testing.T) {
	resolver := landns.SimpleSrvResolver{
		"example.com.": []landns.SrvRecord{
			landns.SrvRecord{Name: landns.Domain("example.com."), Service: "http", Port: 10, Target: landns.Domain("target.local.")},
		},
	}

	if err := resolver.Validate(); err != nil {
		t.Fatalf("failed to validate resolver: %s", err)
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeSRV, false), true, "_http._tcp.example.com. 0 IN SRV 0 0 10 target.local.")

	ResolverTest(t, resolver, landns.NewRequest("empty.example.com.", dns.TypeSRV, false), true)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true)
}

func BenchmarkSimpleSrvResolver(b *testing.B) {
	resolver := landns.SimpleSrvResolver{}

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)
		resolver[host] = []landns.SrvRecord{
			{Name: landns.Domain(host), Service: "http", Port: 10, Target: landns.Domain("example.com.")},
			{Name: landns.Domain(host), Service: "http", Port: 10, Target: landns.Domain("example.com.")},
		}
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeSRV, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(req)
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

	if err := resolverA.Validate(); err != nil {
		t.Fatalf("failed to validate resolverA: %s", err)
	}
	if err := resolverB.Validate(); err != nil {
		t.Fatalf("failed to validate resolverB: %s", err)
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

func BenchmarkResolverSet(b *testing.B) {
	resolver := landns.ResolverSet{}

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)

		resolver = append(resolver, landns.SimpleTxtResolver{
			host: {
				{Name: landns.Domain(host), Text: "hello"},
				{Name: landns.Domain(host), Text: "world"},
			},
		})
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeTXT, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(req)
	}
}
