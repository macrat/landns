package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

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

	defer func() {
		if err := resolver.Close(); err != nil {
			t.Errorf("failed to close: %s", err)
		}
	}()

	if s := resolver.String(); s != "ResolverSet[SimpleResolver[1 domains 1 types 1 records] SimpleResolver[2 domains 1 types 2 records]]" {
		t.Errorf(`unexpected resolver string: "%s"`, s)
	}

	testutil.AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 42 IN A 127.1.1.1", "example.com. 24 IN A 127.1.2.1")
	testutil.AssertResolve(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeA, false), true, "blanktar.jp. 4321 IN A 127.1.3.1")
	testutil.AssertResolve(t, resolver, landns.NewRequest("no.such.com.", dns.TypeA, false), true)
}

func TestResolverSet_ErrorHandling(t *testing.T) {
	response := testutil.EmptyResponseWriter{}
	request := landns.NewRequest("example.com.", dns.TypeA, false)

	errorResolver := testutil.DummyResolver{true, false}
	if err := errorResolver.Resolve(response, request); err == nil {
		t.Fatalf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Fatalf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}

	noErrorResolver := testutil.DummyResolver{false, false}
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

func TestResolverSet_RecursionAvailable(t *testing.T) {
	recursionResolver := testutil.DummyResolver{false, true}
	if recursionResolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}

	nonRecursionResolver := testutil.DummyResolver{false, false}
	if nonRecursionResolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recursion available: %v", nonRecursionResolver.RecursionAvailable())
	}

	resolver := landns.ResolverSet{nonRecursionResolver, recursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}

	resolver = landns.ResolverSet{nonRecursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}
}

func BenchmarkResolverSet(b *testing.B) {
	resolver := make(landns.ResolverSet, 100)
	defer func() {
		if err := resolver.Close(); err != nil {
			b.Errorf("failed to close: %s", err)
		}
	}()

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
	resp := testutil.EmptyResponseWriter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(resp, req)
	}

	b.StopTimer()
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

	defer func() {
		if err := resolver.Close(); err != nil {
			t.Errorf("failed to close: %s", err)
		}
	}()

	testutil.AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 42 IN A 127.1.1.1")
	testutil.AssertResolve(t, resolver, landns.NewRequest("blanktar.jp.", dns.TypeA, false), true, "blanktar.jp. 4321 IN A 127.1.3.1")
	testutil.AssertResolve(t, resolver, landns.NewRequest("no.such.com.", dns.TypeA, false), true)
}

func TestAlternateResolver_ErrorHandling(t *testing.T) {
	response := testutil.EmptyResponseWriter{}
	request := landns.NewRequest("example.com.", dns.TypeA, false)

	errorResolver := testutil.DummyResolver{true, false}
	if err := errorResolver.Resolve(response, request); err == nil {
		t.Fatalf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Fatalf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}

	noErrorResolver := testutil.DummyResolver{false, false}
	if err := noErrorResolver.Resolve(response, request); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	resolver := landns.AlternateResolver{noErrorResolver, errorResolver, noErrorResolver}
	if err := resolver.Resolve(response, request); err == nil {
		t.Errorf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Errorf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}

	resolver = landns.AlternateResolver{errorResolver, noErrorResolver}
	if err := resolver.Resolve(response, request); err == nil {
		t.Errorf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Errorf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}
}

func TestAlternateResolver_RecursionAvailable(t *testing.T) {
	recursionResolver := testutil.DummyResolver{false, true}
	if recursionResolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}

	nonRecursionResolver := testutil.DummyResolver{false, false}
	if nonRecursionResolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recursion available: %v", nonRecursionResolver.RecursionAvailable())
	}

	resolver := landns.AlternateResolver{nonRecursionResolver, recursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}

	resolver = landns.AlternateResolver{nonRecursionResolver, nonRecursionResolver}
	if resolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}
}

func BenchmarkAlternateResolver(b *testing.B) {
	resolver := make(landns.AlternateResolver, 100)
	defer func() {
		if err := resolver.Close(); err != nil {
			b.Errorf("failed to close: %s", err)
		}
	}()

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
	resp := testutil.EmptyResponseWriter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(resp, req)
	}

	b.StopTimer()
}
