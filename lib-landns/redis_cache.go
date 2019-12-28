package landns

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type redisCacheEntry struct {
	Record Record
	Expire time.Time
}

func parseRedisCacheEntry(str string) (redisCacheEntry, error) {
	xs := strings.Split(str, "\n")
	if len(xs) != 2 {
		return redisCacheEntry{}, Error{TypeInternalError, nil, "failed to parse record"}
	}

	i, err := strconv.ParseInt(xs[1], 10, 64)
	if err != nil {
		return redisCacheEntry{}, Error{TypeInternalError, err, "failed to parse record"}
	}

	expire := time.Unix(i, 0)

	record, err := NewRecordWithExpire(xs[0], expire)
	if err != nil {
		return redisCacheEntry{}, Error{TypeInternalError, err, "failed to parse record"}
	}

	return redisCacheEntry{
		Record: record,
		Expire: expire,
	}, nil
}

func (e redisCacheEntry) String() string {
	return fmt.Sprintf("%s\n%d", e.Record, int64(math.Round(float64(e.Expire.UnixNano())/1000/1000/1000)))
}

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

	if err := rc.client.Ping().Err(); err != nil {
		rc.client.Close()
		return RedisCache{}, Error{TypeExternalError, err, "failed to connect to Redis server"}
	}

	return rc, nil
}

// String is description string getter.
func (rc RedisCache) String() string {
	return fmt.Sprintf("RedisCache[%s]", rc.client)
}

// Close is disconnect from Redis server.
func (rc RedisCache) Close() error {
	if err := rc.client.Close(); err != nil {
		return Error{TypeExternalError, err, "failed to close Redis connection"}
	}
	return nil
}

func (rc RedisCache) resolveFromUpstream(w ResponseWriter, r Request, key string) error {
	rc.metrics.CacheHit(r)

	ttl := uint32(math.MaxUint32)
	wh := ResponseWriterHook{
		Writer: w,
		OnAdd: func(record Record) {
			rc.client.RPush(key, redisCacheEntry{
				record,
				time.Now().Add(time.Duration(record.GetTTL()) * time.Second),
			}.String())
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

func (rc RedisCache) resolveFromCache(w ResponseWriter, r Request, cache []string) error {
	rc.metrics.CacheMiss(r)

	for _, str := range cache {
		entry, err := parseRedisCacheEntry(str)
		if err != nil {
			return err
		}

		if err := w.Add(entry.Record); err != nil {
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
		return Error{TypeExternalError, err, "failed to get records"}
	}
	if len(resp) == 0 {
		return rc.resolveFromUpstream(w, r, key)
	}

	return rc.resolveFromCache(w, r, resp)
}

// RecursionAvailable is returns same as upstream.
func (rc RedisCache) RecursionAvailable() bool {
	return rc.upstream.RecursionAvailable()
}
