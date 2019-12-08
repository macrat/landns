package benchmark

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

var (
	targets = []string{"8.8.8.8:53", "1.1.1.1:53"}
)

func NewServer(ctx context.Context, t testutil.FatalFormatter) *net.UDPAddr {
	metrics := landns.NewMetrics("landns")

	addr := testutil.StartDummyDNSServer(ctx, t, landns.AlternateResolver{
		landns.NewSimpleResolver([]landns.Record{}),
		landns.NewLocalCache(landns.NewForwardResolver([]*net.UDPAddr{
			{IP: net.ParseIP("8.8.8.8"), Port: 53},
		}, 100*time.Millisecond, metrics), metrics),
	})

	return addr
}

func BenchmarkDNS(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := NewServer(ctx, b)

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{
			{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		},
	}

	for _, target := range append(targets, server.String()) {
		b.Run(target, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				dns.Exchange(msg, target)
			}
		})
	}
}
