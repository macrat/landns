package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/go-redis/redis"
	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

var (
	redisAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6379}
)

func prepareRedisDB(t testing.TB) {
	t.Helper()

	rds := redis.NewClient(&redis.Options{Addr: redisAddr.String()})
	defer rds.Close()
	if rds.Ping().Err() != nil {
		t.Skip("redis server was not found")
	}
	rds.FlushDB()
}

func TestRedisCache(t *testing.T) {
	prepareRedisDB(t)

	resolver, err := landns.NewRedisCache(redisAddr, 0, "", CacheTestUpstream(t), landns.NewMetrics("landns"))
	if err != nil {
		t.Fatalf("failed to connect redis server: %s", err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	if resolver.String() != fmt.Sprintf("RedisCache[Redis<%s db:0>]", redisAddr) {
		t.Errorf("unexpected string: %s", resolver)
	}

	CacheTest(t, resolver)
}

func TestRedisCache_RecursionAvailable(t *testing.T) {
	prepareRedisDB(t)

	CheckRecursionAvailable(t, func(rs []landns.Resolver) landns.Resolver {
		resolver, err := landns.NewRedisCache(redisAddr, 0, "", landns.ResolverSet(rs), landns.NewMetrics("landns"))
		if err != nil {
			t.Fatalf("failed to connect redis server: %s", err)
		}
		return resolver
	})
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
