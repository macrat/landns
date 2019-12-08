package landns_test

import (
	"net"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

var (
	redisAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6379}
)

type Skipper interface {
	Skip(...interface{})
}

func prepareRedisDB(t Skipper) {
	rds := redis.NewClient(&redis.Options{Addr: redisAddr.String()})
	defer rds.Close()
	if rds.Ping().Err() != nil {
		t.Skip("redis server was not found")
	}
	rds.FlushDB()
}

func TestRedisCache(t *testing.T) {
	prepareRedisDB(t)

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
	resolver, err := landns.NewRedisCache(redisAddr, 0, "", upstream, landns.NewMetrics("landns"))
	if err != nil {
		t.Fatalf("failed to connect redis server: %s", err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	ResolverTest(t, upstream, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 100 IN A 127.1.2.3", "example.com. 10 IN A 127.2.3.4")

	tests := []func(chan struct{}){
		func(ch chan struct{}) {
			ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 100 IN A 127.1.2.3", "example.com. 10 IN A 127.2.3.4")
			time.Sleep(1 * time.Second)
			ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), false, "example.com. 9 IN A 127.1.2.3", "example.com. 9 IN A 127.2.3.4") // Override TTL with minimal TTL
			close(ch)
		},
		func(ch chan struct{}) {
			ResolverTest(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), true, "short.example.com. 1 IN A 127.3.4.5")
			time.Sleep(100 * time.Millisecond)
			ResolverTest(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), false, "short.example.com. 1 IN A 127.3.4.5")
			time.Sleep(1 * time.Second)
			ResolverTest(t, resolver, landns.NewRequest("short.example.com.", dns.TypeA, false), true, "short.example.com. 1 IN A 127.3.4.5")
			close(ch)
		},
		func(ch chan struct{}) {
			ResolverTest(t, resolver, landns.NewRequest("no-cache.example.com.", dns.TypeA, false), true, "no-cache.example.com. 0 IN A 127.4.5.6")
			ResolverTest(t, resolver, landns.NewRequest("no-cache.example.com.", dns.TypeA, false), true, "no-cache.example.com. 0 IN A 127.4.5.6")
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

func BenchmarkRedisCache(b *testing.B) {
	prepareRedisDB(b)

	upstream := landns.NewSimpleResolver([]landns.Record{
		landns.AddressRecord{Name: landns.Domain("example.com."), TTL: 100, Address: net.ParseIP("127.1.2.3")},
	})
	if err := upstream.Validate(); err != nil {
		b.Fatalf("failed to validate upstream resolver: %s", err)
	}
	resolver, err := landns.NewRedisCache(redisAddr, 0, "", upstream, landns.NewMetrics("landns"))
	if err != nil {
		b.Fatalf("failed to connect redis server: %s", err)
	}
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
