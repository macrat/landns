package landns_test

import (
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func TestLocalCache(t *testing.T) {
	upstream := landns.NewSimpleResolver(
		[]landns.Record{
			landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 100, Address: net.ParseIP("127.1.2.3")},
			landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 10, Address: net.ParseIP("127.2.3.4")},
			landns.AddressRecord{Name: landns.Domain("short.example.com."), TTL: 1, Address: net.ParseIP("127.3.4.5")},
			landns.AddressRecord{Name: landns.Domain("no-cache.example.com."), TTL: 0, Address: net.ParseIP("127.4.5.6")},
		},
	)
	if err := upstream.Validate(); err != nil {
		t.Fatalf("failed to validate upstream resolver: %s", err)
	}
	resolver := landns.NewLocalCache(upstream, landns.NewMetrics("landns"))
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	if resolver.String() != "LocalCache[0 domains 0 records]" {
		t.Errorf("unexpected string: %s", resolver)
	}

	AssertResolve(t, upstream, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 100 IN A 127.1.2.3", "example.com. 10 IN A 127.2.3.4")

	tests := []func(chan struct{}){
		func(ch chan struct{}) {
			AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 100 IN A 127.1.2.3", "example.com. 10 IN A 127.2.3.4")
			time.Sleep(1 * time.Second)
			AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), false, "example.com. 99 IN A 127.1.2.3", "example.com. 9 IN A 127.2.3.4")
			close(ch)
		},
		func(ch chan struct{}) {
			AssertResolve(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), true, "short.example.com. 1 IN A 127.3.4.5")
			time.Sleep(100 * time.Millisecond)
			AssertResolve(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), false, "short.example.com. 1 IN A 127.3.4.5")
			time.Sleep(1 * time.Second)
			AssertResolve(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), true, "short.example.com. 1 IN A 127.3.4.5")
			close(ch)
		},
		func(ch chan struct{}) {
			AssertResolve(t, resolver, landns.NewRequest("no-cache.example.com.", dns.TypeA, false), true, "no-cache.example.com. 0 IN A 127.4.5.6")
			AssertResolve(t, resolver, landns.NewRequest("no-cache.example.com.", dns.TypeA, false), true, "no-cache.example.com. 0 IN A 127.4.5.6")
			close(ch)
		},
	}
	waits := make([]chan struct{}, len(tests))
	for i, test := range tests {
		waits[i] = make(chan struct{})
		go test(waits[i])
	}
	for _, ch := range waits {
		<-ch
	}

	if resolver.String() != "LocalCache[2 domains 3 records]" {
		t.Errorf("unexpected string: %s", resolver)
	}
}

func TestLocalCache_RecursionAvailable(t *testing.T) {
	CheckRecursionAvailable(t, func(rs []landns.Resolver) landns.Resolver {
		return landns.NewLocalCache(landns.ResolverSet(rs), landns.NewMetrics("landns"))
	})
}

func BenchmarkLocalCache(b *testing.B) {
	upstream := landns.NewSimpleResolver([]landns.Record{
		landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 100, Address: net.ParseIP("127.1.2.3")},
	})
	if err := upstream.Validate(); err != nil {
		b.Fatalf("failed to validate upstream resolver: %s", err)
	}
	resolver := landns.NewLocalCache(upstream, landns.NewMetrics("landns"))
	defer func() {
		if err := resolver.Close(); err != nil {
			b.Fatalf("failed to close: %s", err)
		}
	}()

	req := landns.NewRequest("example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(NewDummyResponseWriter(), req)
	}

	b.StopTimer()
}
