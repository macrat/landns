package landns_test

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

type DummyResponseWriter struct {
	Records       []landns.Record
	Authoritative bool
}

func NewDummyResponseWriter() *DummyResponseWriter {
	return &DummyResponseWriter{
		Records:       make([]landns.Record, 0, 10),
		Authoritative: true,
	}
}

func (rw *DummyResponseWriter) Add(r landns.Record) error {
	rw.Records = append(rw.Records, r)
	return nil
}

func (rw *DummyResponseWriter) IsAuthoritative() bool {
	return rw.Authoritative
}

func (rw *DummyResponseWriter) SetNoAuthoritative() {
	rw.Authoritative = false
}

type EmptyResponseWriter struct{}

func (rw EmptyResponseWriter) Add(r landns.Record) error {
	return nil
}

func (rw EmptyResponseWriter) IsAuthoritative() bool {
	return true
}

func (rw EmptyResponseWriter) SetNoAuthoritative() {
}

func ResolverTest(t *testing.T, resolver landns.Resolver, request landns.Request, authoritative bool, responses ...string) {
	resp := NewDummyResponseWriter()
	if err := resolver.Resolve(resp, request); err != nil {
		t.Errorf("%s <- %s: failed to resolve: %v", resolver, request, err.Error())
		return
	}

	if resp.Authoritative != authoritative {
		t.Errorf(`%s <- %s: unexpected authoritive of response: expected %v but got %v`, resolver, request, authoritative, resp.Authoritative)
	}

	if len(resp.Records) != len(responses) {
		t.Errorf(`%s <- %s: unexpected resolve response: expected length %d but got %d`, resolver, request, len(responses), len(resp.Records))
		return
	}

	sort.Slice(resp.Records, func(i, j int) bool {
		return strings.Compare(resp.Records[i].String(), resp.Records[j].String()) == 1
	})
	sort.Slice(responses, func(i, j int) bool {
		return strings.Compare(responses[i], responses[j]) == 1
	})

	for i := range responses {
		if resp.Records[i].String() != responses[i] {
			t.Errorf(`%s <- %s: unexpected resolve response: expected "%s" but got "%s"`, resolver, request, responses[i], resp.Records[i])
		}
	}
}

func TestResolverSet(t *testing.T) {
	resolver := landns.ResolverSet{
		landns.SimpleResolver{
			dns.TypeA: {
				"example.com.": {
					landns.AddressRecord{Name: "example.com.", TTL: 42, Address: net.ParseIP("127.1.1.1")},
				},
			},
		},
		landns.SimpleResolver{
			dns.TypeA: {
				"example.com.": {
					landns.AddressRecord{Name: "example.com.", TTL: 24, Address: net.ParseIP("127.1.2.1")},
				},
				"blanktar.jp.": {
					landns.AddressRecord{Name: "blanktar.jp.", TTL: 4321, Address: net.ParseIP("127.1.3.1")},
				},
			},
		},
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 42 A 127.1.1.1", "example.com. 24 A 127.1.2.1")
	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeA, false), true, "blanktar.jp. 4321 A 127.1.3.1")
}

type DummyResolver struct {
	Error     bool
	Recrusion bool
}

func (dr DummyResolver) Resolve(w landns.ResponseWriter, r landns.Request) error {
	if dr.Error {
		return fmt.Errorf("test error")
	} else {
		return nil
	}
}

func (dr DummyResolver) RecursionAvailable() bool {
	return dr.Recrusion
}

func TestResolverSet_ErrorHandling(t *testing.T) {
	response := EmptyResponseWriter{}
	request := landns.NewRequest("example.com.", dns.TypeA, false)

	errorResolver := DummyResolver{true, false}
	if err := errorResolver.Resolve(response, request); err == nil {
		t.Fatalf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Fatalf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}

	noErrorResolver := DummyResolver{false, false}
	if err := noErrorResolver.Resolve(response, request); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	resolver := landns.ResolverSet{noErrorResolver, errorResolver, noErrorResolver}
	if err := resolver.Resolve(response, request); err == nil {
		t.Errorf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Errorf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}
}

func TestResolverSet_RecrusionAvailable(t *testing.T) {
	recursionResolver := DummyResolver{false, true}
	if recursionResolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recrusion available: %v", recursionResolver.RecursionAvailable())
	}

	nonRecursionResolver := DummyResolver{false, false}
	if nonRecursionResolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recrusion available: %v", nonRecursionResolver.RecursionAvailable())
	}

	resolver := landns.ResolverSet{nonRecursionResolver, recursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recrusion available: %v", recursionResolver.RecursionAvailable())
	}

	resolver = landns.ResolverSet{nonRecursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recrusion available: %v", recursionResolver.RecursionAvailable())
	}
}

func BenchmarkResolverSet(b *testing.B) {
	resolver := make(landns.ResolverSet, 100)

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)

		resolver[i] = landns.NewSimpleResolver([]landns.Record{
			landns.AddressRecord{
				Name:    landns.Domain(host),
				Address: net.ParseIP(fmt.Sprintf("127.0.0.%d", i)),
			},
		})
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)
	resp := EmptyResponseWriter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(resp, req)
	}
}

func TestAlternateResolver(t *testing.T) {
	resolver := landns.AlternateResolver{
		landns.SimpleResolver{
			dns.TypeA: {
				"example.com.": {
					landns.AddressRecord{Name: "example.com.", TTL: 42, Address: net.ParseIP("127.1.1.1")},
				},
			},
		},
		landns.SimpleResolver{
			dns.TypeA: {
				"example.com.": {
					landns.AddressRecord{Name: "example.com.", TTL: 24, Address: net.ParseIP("127.1.2.1")},
				},
				"blanktar.jp.": {
					landns.AddressRecord{Name: "blanktar.jp.", TTL: 4321, Address: net.ParseIP("127.1.3.1")},
				},
			},
		},
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 42 A 127.1.1.1")
	ResolverTest(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeA, false), true, "blanktar.jp. 4321 A 127.1.3.1")
}

func TestAlternateResolver_RecrusionAvailable(t *testing.T) {
	recursionResolver := DummyResolver{false, true}
	if recursionResolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recrusion available: %v", recursionResolver.RecursionAvailable())
	}

	nonRecursionResolver := DummyResolver{false, false}
	if nonRecursionResolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recrusion available: %v", nonRecursionResolver.RecursionAvailable())
	}

	resolver := landns.AlternateResolver{nonRecursionResolver, recursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recrusion available: %v", recursionResolver.RecursionAvailable())
	}

	resolver = landns.AlternateResolver{nonRecursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recrusion available: %v", recursionResolver.RecursionAvailable())
	}
}

func BenchmarkAlternateResolver(b *testing.B) {
	resolver := make(landns.AlternateResolver, 100)

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)

		resolver[i] = landns.NewSimpleResolver([]landns.Record{
			landns.AddressRecord{
				Name:    landns.Domain(host),
				Address: net.ParseIP(fmt.Sprintf("127.0.0.%d", i)),
			},
		})
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)
	resp := EmptyResponseWriter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(resp, req)
	}
}
