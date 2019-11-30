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
		t.Errorf("unexcepted authoritative: %v", rc.IsAuthoritative())
	}
	rc.SetNoAuthoritative()
	if rc.IsAuthoritative() != false {
		t.Errorf("unexcepted authoritative: %v", rc.IsAuthoritative())
	}

	if err := rc.Add(landns.AddressRecord{}); err == nil {
		t.Errorf("excepted returns error but got nil")
	} else if err.Error() != "test error" {
		t.Errorf(`unexcepted error: unexcepted "test error" but got "%s"`, err.Error())
	}

	log := make([]landns.Record, 0, 5)
	rc = landns.NewResponseCallback(func(r landns.Record) error {
		log = append(log, r)
		return nil
	})
	for i := 0; i < 5; i++ {
		if len(log) != i {
			t.Errorf("unexcepted log length: excepted %d but got %d", i, len(log))
		}

		text := fmt.Sprintf("test%d", i)
		rc.Add(landns.TxtRecord{Text: text})

		if len(log) != i+1 {
			t.Errorf("unexcepted log length: excepted %d but got %d", i, len(log))
		} else if tr, ok := log[i].(landns.TxtRecord); !ok {
			t.Errorf("unexcepted record type: %#v", log[i])
		} else if tr.Text != text {
			t.Errorf(`unexcepted text: excepted "%s" but got "%s"`, text, tr.Text)
		}
	}
}

func TestMessageBuilder(t *testing.T) {
	builder := landns.NewMessageBuilder(&dns.Msg{})

	if builder.IsAuthoritative() != true {
		t.Errorf("unexcepted authoritative: %v", builder.IsAuthoritative())
	}

	builder.Add(landns.AddressRecord{Name: "example.com.", TTL: 42, Address: net.ParseIP("127.0.1.2")})

	msg := builder.Build()
	if len(msg.Answer) != 1 {
		t.Errorf("unexcepted answer length: excepted 1 but got %d", len(msg.Answer))
	} else if msg.Answer[0].String() != "example.com.\t42\tIN\tA\t127.0.1.2" {
		t.Errorf(`unexcepted answer: excepted "%s" but got "%s"`, "example.com.\t42\tIN\tA\t127.0.1.2", msg.Answer[0].String())
	}
	if msg.Authoritative != true {
		t.Errorf("unexcepted authoritative: %v", builder.IsAuthoritative())
	}

	builder.SetNoAuthoritative()
	builder.Add(landns.AddressRecord{Name: "blanktar.jp.", TTL: 1234, Address: net.ParseIP("127.1.2.3")})

	msg = builder.Build()
	if len(msg.Answer) != 2 {
		t.Errorf("unexcepted answer length: excepted 2 but got %d", len(msg.Answer))
	} else {
		for i, except := range []string{"example.com.\t42\tIN\tA\t127.0.1.2", "blanktar.jp.\t1234\tIN\tA\t127.1.2.3"} {
			if msg.Answer[i].String() != except {
				t.Errorf(`unexcepted answer: excepted "%s" but got "%s"`, except, msg.Answer[i].String())
			}
		}
	}
	if msg.Authoritative != false {
		t.Errorf("unexcepted authoritative: %v", builder.IsAuthoritative())
	}
}
