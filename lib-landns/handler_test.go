package landns_test

import (
	"context"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/logger"
	"github.com/macrat/landns/lib-landns/logger/logtest"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := landns.NewSimpleResolver([]landns.Record{
		landns.AddressRecord{Name: "example.com.", TTL: 123, Address: net.ParseIP("127.0.0.1")},
	})
	if err := resolver.Validate(); err != nil {
		t.Errorf("failed to make resolver: %s", err)
	}

	srv := testutil.StartDNSServer(ctx, t, resolver)

	lt := logtest.Start()
	defer lt.Close()

	srv.Assert(t, dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}, "example.com.\t123\tIN\tA\t127.0.0.1")

	if err := lt.TestAll([]logtest.Entry{}); err != nil {
		t.Error(err)
	}

	srv.Assert(t, dns.Question{Name: "notfound.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET})

	if err := lt.Test([]logtest.Entry{{logger.InfoLevel, "not found", logger.Fields{"name": "notfound.example.com.", "type": "A"}}}); err != nil {
		t.Error(err)
	}

	srv.Assert(t, dns.Question{Name: "notfound.example.com.", Qtype: dns.TypeAAAA, Qclass: dns.ClassINET})

	if err := lt.Test([]logtest.Entry{{logger.InfoLevel, "not found", logger.Fields{"name": "notfound.example.com.", "type": "AAAA"}}}); err != nil {
		t.Error(err)
	}
}

func TestHandler_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := &testutil.DummyResolver{false, false}
	srv := testutil.StartDNSServer(ctx, t, resolver)

	lt := logtest.Start()
	defer lt.Close()

	srv.Assert(t, dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET})

	if err := lt.Test([]logtest.Entry{{logger.InfoLevel, "not found", logger.Fields{"name": "example.com.", "type": "A"}}}); err != nil {
		t.Error(err)
	}

	resolver.Error = true

	srv.Assert(t, dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET})

	if err := lt.Test([]logtest.Entry{{logger.WarnLevel, "failed to resolve", logger.Fields{"reason": "test error", "name": "example.com.", "type": "A"}}}); err != nil {
		t.Error(err)
	}
}
