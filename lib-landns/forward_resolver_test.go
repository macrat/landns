package landns_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func StartDummyDNSServer(ctx context.Context, t *testing.T, resolver landns.Resolver) *net.UDPAddr {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3553}

	server := dns.Server{
		Addr: addr.String(),
		Net:  "udp",
		Handler: landns.Handler{
			Resolver: resolver,
			Metrics:  landns.NewMetrics("landns"),
		},
	}

	go func() {
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			t.Fatalf("failed to serve dummy DNS: %s", err.Error())
		}
	}()

	go func() {
		<-ctx.Done()
		server.Shutdown()
	}()

	return addr
}

func TestForwardResolver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCases := map[landns.Request][]landns.Record{
		landns.NewRequest("example.com.", dns.TypeA, false): {},
		landns.NewRequest("example.com.", dns.TypeA, true): {
			landns.AddressRecord{Name: "example.com.", TTL: 123, Address: net.ParseIP("127.0.0.1")},
			landns.AddressRecord{Name: "example.com.", TTL: 456, Address: net.ParseIP("127.0.0.2")},
		},
		landns.NewRequest("example.com.", dns.TypeAAAA, true): {
			landns.AddressRecord{Name: "example.com.", TTL: 789, Address: net.ParseIP("1::4:2")},
		},
		landns.NewRequest("file.example.com.", dns.TypeCNAME, true): {
			landns.CnameRecord{Name: "file.example.com.", TTL: 123, Target: "example.com."},
		},
		landns.NewRequest("127.0.0.1.in-addr.arpa.", dns.TypePTR, true): {
			landns.PtrRecord{Name: "127.0.0.1.in-addr.arpa.", TTL: 234, Domain: "example.com."},
		},
		landns.NewRequest("example.com.", dns.TypeSRV, true): {
			landns.SrvRecord{
				Name:     "example.com.",
				TTL:      234,
				Service:  "http",
				Proto:    "tcp",
				Priority: 1,
				Weight:   2,
				Port:     3,
				Target:   "file.example.com.",
			},
		},
		landns.NewRequest("example.com.", dns.TypeTXT, true): {
			landns.TxtRecord{Name: "example.com.", TTL: 234, Text: "hello world"},
		},
	}
	records := []landns.Record{}
	for _, rs := range testCases {
		records = append(records, rs...)
	}
	addr := StartDummyDNSServer(ctx, t, landns.NewSimpleResolver(records))

	resolver := landns.NewForwardResolver([]*net.UDPAddr{addr}, 1*time.Second, landns.NewMetrics("landns"))

	for request, records := range testCases {
		rs := []string{}
		for _, r := range records {
			rs = append(rs, r.String())
		}
		ResolverTest(t, resolver, request, !request.RecursionDesired, rs...)
	}
}
