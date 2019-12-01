package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func TestResponseCallback(t *testing.T) {
	rc := landns.NewResponseCallback(func(r landns.Record) error {
		return fmt.Errorf("test error")
	})

	if rc.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: %v", rc.IsAuthoritative())
	}
	rc.SetNoAuthoritative()
	if rc.IsAuthoritative() != false {
		t.Errorf("unexpected authoritative: %v", rc.IsAuthoritative())
	}

	if err := rc.Add(landns.AddressRecord{}); err == nil {
		t.Errorf("expected returns error but got nil")
	} else if err.Error() != "test error" {
		t.Errorf(`unexpected error: unexpected "test error" but got "%s"`, err.Error())
	}

	log := make([]landns.Record, 0, 5)
	rc = landns.NewResponseCallback(func(r landns.Record) error {
		log = append(log, r)
		return nil
	})
	for i := 0; i < 5; i++ {
		if len(log) != i {
			t.Errorf("unexpected log length: expected %d but got %d", i, len(log))
		}

		text := fmt.Sprintf("test%d", i)
		rc.Add(landns.TxtRecord{Text: text})

		if len(log) != i+1 {
			t.Errorf("unexpected log length: expected %d but got %d", i, len(log))
		} else if tr, ok := log[i].(landns.TxtRecord); !ok {
			t.Errorf("unexpected record type: %#v", log[i])
		} else if tr.Text != text {
			t.Errorf(`unexpected text: expected "%s" but got "%s"`, text, tr.Text)
		}
	}
}

func TestMessageBuilder(t *testing.T) {
	builder := landns.NewMessageBuilder(&dns.Msg{}, true)

	if builder.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: %v", builder.IsAuthoritative())
	}

	builder.Add(landns.AddressRecord{Name: "example.com.", TTL: 42, Address: net.ParseIP("127.0.1.2")})

	msg := builder.Build()
	if len(msg.Answer) != 1 {
		t.Errorf("unexpected answer length: expected 1 but got %d", len(msg.Answer))
	} else if msg.Answer[0].String() != "example.com.\t42\tIN\tA\t127.0.1.2" {
		t.Errorf(`unexpected answer: expected "%s" but got "%s"`, "example.com.\t42\tIN\tA\t127.0.1.2", msg.Answer[0].String())
	}
	if msg.Authoritative != true {
		t.Errorf("unexpected authoritative: %v", msg.Authoritative)
	}
	if msg.RecursionAvailable != true {
		t.Errorf("unexpected recurtion available: %v", msg.RecursionAvailable)
	}

	builder.SetNoAuthoritative()
	builder.Add(landns.AddressRecord{Name: "blanktar.jp.", TTL: 1234, Address: net.ParseIP("127.1.2.3")})

	msg = builder.Build()
	if len(msg.Answer) != 2 {
		t.Errorf("unexpected answer length: expected 2 but got %d", len(msg.Answer))
	} else {
		for i, expect := range []string{"example.com.\t42\tIN\tA\t127.0.1.2", "blanktar.jp.\t1234\tIN\tA\t127.1.2.3"} {
			if msg.Answer[i].String() != expect {
				t.Errorf(`unexpected answer: expected "%s" but got "%s"`, expect, msg.Answer[i].String())
			}
		}
	}
	if msg.Authoritative != false {
		t.Errorf("unexpected authoritative: %v", msg.RecursionAvailable)
	}
	if msg.RecursionAvailable != true {
		t.Errorf("unexpected recurtion available: %v", msg.RecursionAvailable)
	}

	builder = landns.NewMessageBuilder(&dns.Msg{}, false)
	msg = builder.Build()
	if msg.RecursionAvailable != false {
		t.Errorf("unexpected recurtion available: %v", msg.RecursionAvailable)
	}
}
