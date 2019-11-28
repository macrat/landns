package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func BenchmarkSqliteResolver(b *testing.B) {
	metrics := landns.NewMetrics("landns")
	resolver, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		b.Fatalf("failed to make sqlite resolver: %s", err.Error())
	}

	config := landns.AddressesConfig{}

	for i := 0; i < 100; i++ {
		config[landns.Domain(fmt.Sprintf("host%d.example.com.", i))] = []landns.AddressRecordConfig{
			{Address: net.ParseIP("127.1.2.3")},
			{Address: net.ParseIP("127.1.2.4")},
		}
	}

	resolver.UpdateAddresses(config)

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(req)
	}
}
