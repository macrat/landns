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

	addr := testutil.StartDummyDNSServer(ctx, t, resolver)

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{
			{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		},
	}

	in, err := dns.Exchange(msg, addr.String())
	if err != nil {
		t.Errorf("failed to resolve: %s", err)
	} else if len(in.Answer) != 1 {
		t.Errorf("unexpected answer length: %d: %#v", len(in.Answer), in.Answer)
	} else if in.Answer[0].String() != "example.com.\t123\tIN\tA\t127.0.0.1" {
		t.Errorf("unexpected answer: %s", in.Answer[0])
	}

	msg = &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{
			{Name: "notfound.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		},
	}

	in, err = dns.Exchange(msg, addr.String())
	if err != nil {
		t.Errorf("failed to resolve: %s", err)
	} else if len(in.Answer) != 0 {
		t.Errorf("unexpected answer length: %d: %#v", len(in.Answer), in.Answer)
	}
}

func TestHandler_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolver := &testutil.DummyResolver{false, false}
	addr := testutil.StartDummyDNSServer(ctx, t, resolver)

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{
			{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		},
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	in, err := dns.Exchange(msg, addr.String())
	if err != nil {
		t.Errorf("failed to resolve: %s", err)
	} else if len(in.Answer) != 0 {
		t.Errorf("unexpected answer length: %d: %#v", len(in.Answer), in.Answer)
	}

	if len(buf.String()) != 0 {
		t.Errorf("unexpected log: %s", buf.String())
	}

	resolver.Error = true

	in, err = dns.Exchange(msg, addr.String())
	if err != nil {
		t.Errorf("failed to resolve: %s", err)
	} else if len(in.Answer) != 0 {
		t.Errorf("unexpected answer length: %d: %#v", len(in.Answer), in.Answer)
	}

	if regexp.MustCompile(`20[0-9]{2}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} test error\r`).MatchString(buf.String()) {
		t.Errorf("unexpected log:\n%s", buf.String())
	}
}
