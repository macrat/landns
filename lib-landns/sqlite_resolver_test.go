package landns_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns"
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

func TestSqliteResolver_Parallel(t *testing.T) {
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
