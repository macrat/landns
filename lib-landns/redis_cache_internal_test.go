package landns

import (
	"fmt"
	"testing"
	"time"
)

func TestParseRedisCacheEntry(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			Record string
			Offset time.Duration
			Expect string
		}{
			{
				"example.com. 600 IN A 127.0.0.1",
				42 * time.Second,
				"example.com. 42 IN A 127.0.0.1",
			},
		}

		for _, tt := range tests {
			e, err := parseRedisCacheEntry(fmt.Sprintf("%s\n%d", tt.Record, time.Now().Add(tt.Offset+500*time.Millisecond).Unix()))

			if err != nil {
				t.Errorf("failed to parse cache entry: %s", err)
				continue
			}

			if e.Record.String() != tt.Expect {
				t.Errorf("unexpected record string:\nexpected: %#v\nbut got:  %#v", tt.Expect, e.Record.String())
			}
		}
	})

	t.Run("error", func(t *testing.T) {
		tests := []struct {
			Entry string
			Error string
		}{
			{
				"example.com. 600 IN A 127.0.0.1\n12345",
				"failed to parse record: expire can't be past time.",
			},
			{
				"hello world\n4294967295",
				"failed to parse record: failed to parse record: dns: not a TTL: \"world\" at line: 1:11",
			},
			{
				"example.com. 600 IN A 127.0.0.1\n",
				"failed to parse record: strconv.ParseInt: parsing \"\": invalid syntax",
			},
			{
				"example.com. 600 IN A 127.0.0.1",
				"failed to parse record",
			},
		}

		for _, tt := range tests {
			_, err := parseRedisCacheEntry(tt.Entry)

			if err == nil {
				t.Errorf("expected error but got nil")
			} else if err.Error() != tt.Error {
				t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", tt.Error, err.Error())
			}
		}
	})
}
