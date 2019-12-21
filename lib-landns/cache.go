package landns

import (
	"time"
)

type cacheEntry struct {
	Record  Record
	Created time.Time
	Expire  time.Time
}
