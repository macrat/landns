package landns_test

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"regexp"
	"testing"

	"github.com/macrat/landns/lib-landns"
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

	addr := StartDummyDNSServer(ctx, t, resolver)

	AssertExchange(t, addr, []dns.Question{
		{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}, "example.com.\t123\tIN\tA\t127.0.0.1")

	AssertExchange(t, addr, []dns.Question{
		{Name: "notfound.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	})
}

func TestHandler_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := &DummyResolver{false, false}
	addr := StartDummyDNSServer(ctx, t, resolver)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	AssertExchange(t, addr, []dns.Question{
		{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	})

	if len(buf.String()) != 0 {
		t.Errorf("unexpected log: %s", buf.String())
	}

	resolver.Error = true

	AssertExchange(t, addr, []dns.Question{
		{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	})

	if regexp.MustCompile(`20[0-9]{2}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} test error\r`).MatchString(buf.String()) {
		t.Errorf("unexpected log:\n%s", buf.String())
	}
}
