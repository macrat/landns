package landns

import (
	"fmt"
	"math"
	"net"
	"time"

	"github.com/go-redis/redis"
	"github.com/miekg/dns"
)

// RedisCache is redis cache manager for Resolver.
type RedisCache struct {
	client   *redis.Client
	upstream Resolver
	metrics  *Metrics
}

// NewRedisCache is constructor of RedisCache.
//
// RedisCache will make connection to the Redis server. So you have to ensure to call RedisCache.Close.
func NewRedisCache(addr *net.TCPAddr, database int, password string, upstream Resolver, metrics *Metrics) (RedisCache, error) {
	rc := RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr.String(),
			Password: password,
			DB:       database,
		}),
		upstream: upstream,
		metrics:  metrics,
	}
	return rc, rc.client.Ping().Err()
}

// Close is disconnect from Redis server.
func (rc RedisCache) Close() error {
	return rc.client.Close()
}

func (rc RedisCache) resolveFromUpstream(w ResponseWriter, r Request, key string) error {
	rc.metrics.CacheHit(r)

	ttl := uint32(math.MaxUint32)
	wh := ResponseWriterHook{
		Writer: w,
		OnAdd: func(record Record) {
			rc.client.RPush(key, record.String())
			if ttl > record.GetTTL() {
				ttl = record.GetTTL()
			}
		},
	}

	if err := rc.upstream.Resolve(wh, r); err != nil {
		rc.client.Del(key)
		return err
	}

	if ttl == 0 {
		rc.client.Del(key)
	} else {
		rc.client.Expire(key, time.Duration(ttl)*time.Second)
	}
	return nil
}

func (rc RedisCache) resolveFromCache(w ResponseWriter, r Request, cache []string, ttl time.Duration) error {
	rc.metrics.CacheMiss(r)

	for _, str := range cache {
		rr, err := dns.NewRR(str)
		if err != nil {
			return err
		}

		rr.Header().Ttl = uint32(ttl.Seconds())

		if record, err := NewRecordFromRR(rr); err != nil {
			return err
		} else if err := w.Add(record); err != nil {
			return err
		}

		w.SetNoAuthoritative()
	}

	return nil
}

// Resolve is resolver using cache or the upstream resolver.
func (rc RedisCache) Resolve(w ResponseWriter, r Request) error {
	key := fmt.Sprintf("%s:%s", r.QtypeString(), r.Name)

	resp, err := rc.client.LRange(key, 0, -1).Result()
	if err != nil {
		return err
	}
	if len(resp) == 0 {
		return rc.resolveFromUpstream(w, r, key)
	}

	ttl, err := rc.client.TTL(key).Result()
	if err != nil {
		return err
	}

	return rc.resolveFromCache(w, r, resp, ttl)
}

// RecursionAvailable is returns same as upstream.
func (rc RedisCache) RecursionAvailable() bool {
	return rc.upstream.RecursionAvailable()
}
