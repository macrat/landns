package landns_test

import (
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func CreateSqliteResolver(t testing.TB) *landns.SqliteResolver {
	t.Helper()

	metrics := landns.NewMetrics("landns")
	resolver, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %s", err.Error())
	}

	return resolver
}

func TestSqliteResolver(t *testing.T) {
	t.Parallel()

	resolver := CreateSqliteResolver(t)
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	if s := resolver.String(); s != "SqliteResolver[:memory:]" {
		t.Errorf(`unexpected string: expected "SqliteResolver[:memory:]" but got %#v`, s)
	}

	DynamicResolverTest(t, resolver)
}

func TestSqliteResolver_Volatile(t *testing.T) {
	t.Parallel()

	resolver := CreateSqliteResolver(t)
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	records, err := landns.NewDynamicRecordSet(`
		fixed.example.com. 100 IN TXT "fixed"
		long.example.com. 100 IN TXT "long" ; Volatile
		short.example.com. 1 IN TXT "short" ; Volatile
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	time.Sleep(1500 * time.Millisecond)

	rs, err := resolver.Records()
	if err != nil {
		t.Errorf("failed to get records: %s", err)
	}
	AssertDynamicRecordSet(t, "volatile records", []string{
		`fixed.example.com. 100 IN TXT "fixed" ; ID:1`,
		`long.example.com. 98 IN TXT "long" ; ID:2`,
	}, rs)

	AssertResolve(t, resolver, landns.NewRequest("long.example.com.", dns.TypeTXT, false), true, `long.example.com. 98 IN TXT "long"`)
}

func TestSqliteResolver_Parallel(t *testing.T) {
	t.Parallel()

	resolver := CreateSqliteResolver(t)
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	ParallelResolveTest(t, resolver)
}

func BenchmarkSqliteResolver(b *testing.B) {
	resolver := CreateSqliteResolver(b)
	defer func() {
		if err := resolver.Close(); err != nil {
			b.Fatalf("failed to close: %s", err)
		}
	}()

	DynamicResolverBenchmark(b, resolver)
}
