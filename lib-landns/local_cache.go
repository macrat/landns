package landns

import (
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type localCacheEntry struct {
	Record  Record
	Created time.Time
	Expire  time.Time
}

type LocalCache struct {
	mutex    sync.Mutex
	entries  map[uint16]map[Domain][]localCacheEntry
	invoke   chan struct{}
	closer   chan struct{}
	upstream Resolver
}

func NewLocalCache(upstream Resolver) *LocalCache {
	lc := &LocalCache{
		entries:  make(map[uint16]map[Domain][]localCacheEntry),
		invoke:   make(chan struct{}, 100),
		closer:   make(chan struct{}),
		upstream: upstream,
	}

	for _, t := range []uint16{dns.TypeA, dns.TypeNS, dns.TypeCNAME, dns.TypePTR, dns.TypeMX, dns.TypeTXT, dns.TypeAAAA, dns.TypeSRV} {
		lc.entries[t] = make(map[Domain][]localCacheEntry)
	}

	go lc.manage()

	return lc
}

func (lc *LocalCache) String() string {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	domains := make(map[Domain]struct{})
	records := 0
	for _, xs := range lc.entries {
		for name, x := range xs {
			domains[name] = struct{}{}
			records += len(x)
		}
	}

	return fmt.Sprintf("LocalCache[%d domains %d records]", len(domains), records)
}

func (lc *LocalCache) Close() error {
	close(lc.closer)
	close(lc.invoke)
	return nil
}

func (lc *LocalCache) manageTask() (next time.Duration) {
	next = 10 * time.Second

	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	for _, domains := range lc.entries {
		for name, entries := range domains {
			sweep := false

			for _, entry := range entries {
				delta := entry.Expire.Sub(time.Now())
				if delta < 1 {
					sweep = true
					break
				} else if next > delta {
					next = delta
				}
			}

			if sweep {
				delete(domains, name)
			}
		}
	}

	return next
}

func (lc *LocalCache) manage() {
	for {
		lc.manageTask()

		select {
		case <-time.After(lc.manageTask()):
		case <-lc.invoke:
		case <-lc.closer:
			return
		}
	}
}

func (lc *LocalCache) add(r Record) {
	if r.GetTTL() == 0 {
		return
	}

	if _, ok := lc.entries[r.GetQtype()][r.GetName()]; !ok {
		lc.entries[r.GetQtype()][r.GetName()] = []localCacheEntry{
			{
				Record:  r,
				Created: time.Now(),
				Expire:  time.Now().Add(time.Duration(r.GetTTL()) * time.Second),
			},
		}
	} else {
		lc.entries[r.GetQtype()][r.GetName()] = append(lc.entries[r.GetQtype()][r.GetName()], localCacheEntry{
			Record:  r,
			Created: time.Now(),
			Expire:  time.Now().Add(time.Duration(r.GetTTL()) * time.Second),
		})
	}

	lc.invoke <- struct{}{}
}

func (lc *LocalCache) resolveFromUpstream(w ResponseWriter, r Request) error {
	wh := ResponseWriterHook{
		Writer: w,
		OnAdd:  lc.add,
	}

	return lc.upstream.Resolve(wh, r)
}

func (lc *LocalCache) resolveFromCache(w ResponseWriter, records []localCacheEntry) error {
	w.SetNoAuthoritative()

	now := time.Now()

	for _, cache := range records {
		rr, err := cache.Record.ToRR()
		if err != nil {
			return err
		}

		rr.Header().Ttl -= uint32(now.Sub(cache.Created).Seconds())

		record, err := NewRecordFromRR(rr)
		if err != nil {
			return err
		}

		if err := w.Add(record); err != nil {
			return err
		}
	}

	return nil
}

func (lc *LocalCache) Resolve(w ResponseWriter, r Request) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	records, ok := lc.entries[r.Qtype][Domain(r.Name)]
	if !ok {
		return lc.resolveFromUpstream(w, r)
	}

	for _, cache := range records {
		if cache.Expire.Sub(time.Now()) < 1 {
			delete(lc.entries[r.Qtype], Domain(r.Name))
			return lc.resolveFromUpstream(w, r)
		}
	}

	return lc.resolveFromCache(w, records)
}

func (lc *LocalCache) RecursionAvailable() bool {
	return lc.upstream.RecursionAvailable()
}
