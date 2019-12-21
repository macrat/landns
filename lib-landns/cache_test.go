package landns_test

import (
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func CacheTestUpstream(t testing.TB) landns.Resolver {
	upstream := landns.NewSimpleResolver(
		[]landns.Record{
			landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 100, Address: net.ParseIP("127.1.2.3")},
			landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 10, Address: net.ParseIP("127.2.3.4")},
			landns.AddressRecord{Name: landns.Domain("short.example.com."), TTL: 10, Address: net.ParseIP("127.3.4.5")},
			landns.AddressRecord{Name: landns.Domain("short.example.com."), TTL: 2, Address: net.ParseIP("127.4.5.6")},
			landns.AddressRecord{Name: landns.Domain("no-cache.example.com."), TTL: 0, Address: net.ParseIP("127.5.6.7")},
			landns.TxtRecord{Name: landns.Domain("example.com."), TTL: 100, Text: "hello world"},
		},
	)
	if err := upstream.Validate(); err != nil {
		t.Fatalf("failed to validate upstream resolver: %s", err)
	}

	AssertResolve(t, upstream, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 100 IN A 127.1.2.3", "example.com. 10 IN A 127.2.3.4")

	return upstream
}

func CacheTest(t *testing.T, resolver landns.Resolver) {
	tests := []func(chan struct{}){
		func(ch chan struct{}) {
			AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 100 IN A 127.1.2.3", "example.com. 10 IN A 127.2.3.4")
			time.Sleep(1 * time.Second)
			AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), false, "example.com. 99 IN A 127.1.2.3", "example.com. 9 IN A 127.2.3.4")
			close(ch)
		},
		func(ch chan struct{}) {
			AssertResolve(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), true, "short.example.com. 10 IN A 127.3.4.5", "short.example.com. 2 IN A 127.4.5.6")
			time.Sleep(1 * time.Second)
			AssertResolve(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), false, "short.example.com. 9 IN A 127.3.4.5", "short.example.com. 1 IN A 127.4.5.6")
			time.Sleep(1200 * time.Millisecond)
			AssertResolve(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), true, "short.example.com. 10 IN A 127.3.4.5", "short.example.com. 2 IN A 127.4.5.6")
			close(ch)
		},
		func(ch chan struct{}) {
			AssertResolve(t, resolver, landns.NewRequest("no-cache.example.com.", dns.TypeA, false), true, "no-cache.example.com. 0 IN A 127.5.6.7")
			AssertResolve(t, resolver, landns.NewRequest("no-cache.example.com.", dns.TypeA, false), true, "no-cache.example.com. 0 IN A 127.5.6.7")
			close(ch)
		},
		func(ch chan struct{}) {
			AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true, "example.com. 100 IN TXT \"hello world\"")
			time.Sleep(1 * time.Second)
			AssertResolve(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), false, "example.com. 99 IN TXT \"hello world\"")
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
}
