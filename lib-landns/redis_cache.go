package landns

import (
	"fmt"
	"math"
	"net"
	"time"

	"github.com/gomodule/redigo/redis"
)

// RedisCache is redis cache manager for Resolver.
type RedisCache struct {
	addr     net.Addr
	pool     *redis.Pool
	upstream Resolver
	metrics  *Metrics
}

/*
NewRedisCache is constructor of RedisCache.

RedisCache will make connection to the Redis server. So you have to ensure to call RedisCache.Close.
*/
func NewRedisCache(addr net.Addr, database int, password string, upstream Resolver, metrics *Metrics) (RedisCache, error) {
	pool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial(
			addr.Network(),
			addr.String(),
			redis.DialDatabase(database),
			redis.DialPassword(password),
		)
	}, 5)

	con := pool.Get()
	defer con.Close()
	if err := con.Err(); err != nil {
		return RedisCache{}, Error{TypeExternalError, err, "failed to connect to Redis server"}
	}

	return RedisCache{
		addr: addr,
		pool: pool,
		upstream: upstream,
		metrics: metrics,
	}, nil
}

// String is description string getter.
func (rc RedisCache) String() string {
	return fmt.Sprintf("RedisCache[%s, %s]", rc.addr, rc.upstream)
}

// Close is disconnect from Redis server.
func (rc RedisCache) Close() error {
	if err := rc.pool.Close(); err != nil {
		return Error{TypeExternalError, err, "failed to close Redis connection"}
	}
	return nil
}

func (rc RedisCache) resolveFromUpstream(w ResponseWriter, r Request, key string) error {
	rc.metrics.CacheHit(r)

	conn := rc.pool.Get()
	defer conn.Close()
	if err := conn.Send("MULTI"); err != nil {
		return err
	}
	rollback := func() error {
		return conn.Send("DISCARD")
	}
	commit := func() error {
		return conn.Send("EXEC")
	}

	ttl := uint32(math.MaxUint32)
	wh := ResponseWriterHook{
		Writer: w,
		OnAdd: func(record Record) error {
			if ttl > record.GetTTL() {
				ttl = record.GetTTL()
			}

			rr, err := record.ToRR()
			if err != nil {
				rollback()
				return err
			}

			rec := VolatileRecord{
				RR: rr,
				Expire: time.Now().Add(time.Duration(record.GetTTL()) * time.Second),
			}

			if err := conn.Send("RPUSH", key, rec.String()); err != nil {
				rollback()
				return err
			}

			return nil
		},
	}

	if err := rc.upstream.Resolve(wh, r); err != nil {
		rollback()
		return err
	}

	if ttl > 0 {
		if err := conn.Send("EXPIRE", key, ttl); err != nil {
			return nil
		}
		return commit()
	}
	return rollback()
}

func (rc RedisCache) resolveFromCache(w ResponseWriter, r Request, cache []string) error {
	rc.metrics.CacheMiss(r)

	for _, str := range cache {
		entry, err := NewVolatileRecord(str)
		if err != nil {
			return err
		}

		if rec, err := entry.Record(); err != nil {
			continue
		} else if err := w.Add(rec); err != nil {
			return err
		}

		w.SetNoAuthoritative()
	}

	return nil
}

// Resolve is resolver using cache or the upstream resolver.
func (rc RedisCache) Resolve(w ResponseWriter, r Request) error {
	key := fmt.Sprintf("%s:%s", r.QtypeString(), r.Name)

	conn := rc.pool.Get()
	defer conn.Close()

	resp, err := redis.Strings(conn.Do("LRANGE", key, 0, -1))
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
