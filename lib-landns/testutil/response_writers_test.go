package testutil_test

import (
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
)

func TestDummyResponseWriter(t *testing.T) {
	t.Parallel()

	w := testutil.NewDummyResponseWriter()

	if w.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: expected true but got false")
	}
	w.SetNoAuthoritative()
	if w.IsAuthoritative() != false {
		t.Errorf("unexpected authoritative: expected false but got true")
	}

	records := []landns.Record{
		landns.AddressRecord{"example.com.", 42, net.ParseIP("127.1.2.3")},
		landns.AddressRecord{"example.com.", 123, net.ParseIP("127.9.8.7")},
	}
	for _, r := range records {
		if err := w.Add(r); err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
	}

	for i := range w.Records {
		if records[i].String() != w.Records[i].String() {
			t.Errorf("unexpected record: expected %#v but got %#v", records[i], w.Records[i])
		}
	}
}

func TestEmptyResponseWriter(t *testing.T) {
	t.Parallel()

	w := testutil.EmptyResponseWriter{}

	if w.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: expected true but got false")
	}
	w.SetNoAuthoritative()
	if w.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: expected true but got false")
	}

	if err := w.Add(landns.AddressRecord{"example.com.", 42, net.ParseIP("127.1.2.3")}); err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}
}
