package landns_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"go.etcd.io/etcd/integration"
)

func CreateEtcdResolver(t testing.TB) (*landns.EtcdResolver, []string, func()) {
	clus := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 1, SkipCreatingClient: true})

	addrs := make([]string, len(clus.Members))
	for i, m := range clus.Members {
		addrs[i] = m.GRPCAddr()
	}

	resolver, err := landns.NewEtcdResolver(addrs, "/landns", time.Second, landns.NewMetrics("metrics"))
	if err != nil {
		t.Fatalf("failed to make etcd resolver: %s", err)
	}

	return resolver, addrs, func() {
		clus.Terminate(t)
		if err := resolver.Close(); err != nil {
			t.Errorf("failed to close: %s", err)
		}
	}
}

func TestEtcdResolver(t *testing.T) {
	t.Parallel()

	resolver, addrs, closer := CreateEtcdResolver(t)
	defer closer()

	name := fmt.Sprintf("EtcdResolver[%s]", addrs[0])
	if s := resolver.String(); s != name {
		t.Errorf(`unexpected string: expected %#v but got %#v`, name, s)
	}

	DynamicResolverTest(t, resolver)
}

func TestEtcdResolver_Parallel(t *testing.T) {
	t.Parallel()

	resolver, _, closer := CreateEtcdResolver(t)
	defer closer()

	ParallelResolveTest(t, resolver)
}

func BenchmarkEtcdResolver(b *testing.B) {
	resolver, _, closer := CreateEtcdResolver(b)
	defer closer()

	DynamicResolverBenchmark(b, resolver)
}
