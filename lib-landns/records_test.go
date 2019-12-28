package landns_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func TestDomain_Validate(t *testing.T) {
	a := landns.Domain("")
	if err := a.Validate(); err == nil {
		t.Errorf("failed to empty domain validation: <nil>")
	} else if err.Error() != `invalid domain: ""` {
		t.Errorf("failed to empty domain validation: %#v", err.Error())
	}

	b := landns.Domain("..")
	if err := b.Validate(); err == nil {
		t.Errorf("failed to invalid domain validation: <nil>")
	} else if err.Error() != `invalid domain: ".."` {
		t.Errorf("failed to invalid domain validation: %#v", err.Error())
	}

	c := landns.Domain("example.com.")
	if err := c.Validate(); err != nil {
		t.Errorf("failed to valid domain validation: %#v", err.Error())
	}

	d := landns.Domain("example.com")
	if err := d.Validate(); err != nil {
		t.Errorf("failed to valid domain validation: %#v", err.Error())
	}
}

func TestDomain_Encoding(t *testing.T) {
	var d landns.Domain

	for input, expect := range map[string]string{"": ".", "example.com": "example.com.", "blanktar.jp.": "blanktar.jp."} {
		if err := (&d).UnmarshalText([]byte(input)); err != nil {
			t.Errorf("failed to unmarshal: %s: %s", input, err)
		} else if result, err := d.MarshalText(); err != nil {
			t.Errorf("failed to marshal: %s: %s", input, err)
		} else if string(result) != expect {
			t.Errorf("unexpected marshal result: expected %s but got %s", expect, string(result))
		}
	}

	if err := (&d).UnmarshalText([]byte("example.com..")); err == nil {
		t.Errorf("expected error but got nil")
	} else if err.Error() != `invalid domain: "example.com.."` {
		t.Errorf(`unexpected error: expected 'invalid domain: "example.com.."' but got '%s'`, err)
	}
}

func TestDomain_ToPath(t *testing.T) {
	tests := []struct {
		Input  landns.Domain
		Expect string
	}{
		{"example.com.", "/com/example"},
		{"", "/"},
		{"a.b.c.d", "/d/c/b/a"},
	}

	for _, tt := range tests {
		if p := tt.Input.ToPath(); p != tt.Expect {
			t.Errorf("unexpected path:\nexpected: %s\nbut got:  %s", tt.Expect, p)
		}
	}
}

func TestNewRecordWithExpire(t *testing.T) {
	tests := []struct {
		String string
		Offset time.Duration
		Expect string
		Error  string
	}{
		{"example.com. 600 IN A 127.0.0.1", 42 * time.Second, "example.com. 42 IN A 127.0.0.1", ""},
		{"example.com. 500 IN A 127.0.0.2", 42 * time.Second, "example.com. 42 IN A 127.0.0.2", ""},
		{"example.com. 400 IN A 127.0.0.3", 400 * time.Second, "example.com. 400 IN A 127.0.0.3", ""},
		{"example.com. 300 IN A 127.0.0.3", time.Millisecond, "example.com. 0 IN A 127.0.0.3", ""},
		{"example.com. 400 IN A 127.0.0.3", -time.Second, "", "expire can't be past time."},
	}

	for _, tt := range tests {
		r, err := landns.NewRecordWithExpire(tt.String, time.Now().Add(tt.Offset))

		if err != nil {
			if tt.Error == "" {
				t.Errorf("failed to parse record: %s", err)
			} else if err.Error() != tt.Error {
				t.Errorf("unexpected error:\nexpected: %#v\nbut got:  %#v", tt.Error, err.Error())
			}
			continue
		}

		if r.String() != tt.Expect {
			t.Errorf("unexpected parse result:\nexpected: %#v\nbut got:  %#v", tt.Expect, r.String())
		}
	}
}

func TestRecords(t *testing.T) {
	tests := []struct {
		Record landns.Record
		String string
		Qtype  uint16
		TTL    uint32
	}{
		{
			landns.AddressRecord{Name: "a.example.com.", TTL: 10, Address: net.ParseIP("127.0.0.1")},
			"a.example.com. 10 IN A 127.0.0.1",
			dns.TypeA,
			10,
		},
		{
			landns.AddressRecord{Name: "aaaa.example.com.", TTL: 20, Address: net.ParseIP("4::2")},
			"aaaa.example.com. 20 IN AAAA 4::2",
			dns.TypeAAAA,
			20,
		},
		{
			landns.NsRecord{Name: "ns.example.com.", Target: "example.com."},
			"ns.example.com. IN NS example.com.",
			dns.TypeNS,
			0,
		},
		{
			landns.CnameRecord{Name: "cname.example.com.", TTL: 40, Target: "example.com."},
			"cname.example.com. 40 IN CNAME example.com.",
			dns.TypeCNAME,
			40,
		},
		{
			landns.PtrRecord{Name: "1.0.0.127.in-addr.arpa.", TTL: 50, Domain: "ptr.example.com."},
			"1.0.0.127.in-addr.arpa. 50 IN PTR ptr.example.com.",
			dns.TypePTR,
			50,
		},
		{
			landns.MxRecord{Name: "mx.example.com.", TTL: 60, Preference: 42, Target: "example.com."},
			"mx.example.com. 60 IN MX 42 example.com.",
			dns.TypeMX,
			60,
		},
		{
			landns.TxtRecord{Name: "txt.example.com.", TTL: 70, Text: "hello world"},
			"txt.example.com. 70 IN TXT \"hello world\"",
			dns.TypeTXT,
			70,
		},
		{
			landns.SrvRecord{Name: "_web._tcp.srv.example.com.", TTL: 80, Priority: 11, Weight: 22, Port: 33, Target: "example.com."},
			"_web._tcp.srv.example.com. 80 IN SRV 11 22 33 example.com.",
			dns.TypeSRV,
			80,
		},
	}

	for _, tt := range tests {
		if err := tt.Record.Validate(); err != nil {
			t.Errorf("failed to validate: %s", err)
		}
		if s := tt.Record.String(); s != tt.String {
			t.Errorf("failed to convert to string:\nexpected: %s\nbut got:  %s", tt.String, s)
		}
		if q := tt.Record.GetQtype(); q != tt.Qtype {
			t.Errorf("unexpected qtype: expected %d but got %d", tt.Qtype, q)
		}
		if ttl := tt.Record.GetTTL(); ttl != tt.TTL {
			t.Errorf("unexpected ttl: expected %d but got %d", tt.TTL, ttl)
		}

		rr1, err := tt.Record.ToRR()
		if err != nil {
			t.Errorf("failed to convert to dns.RR: %s", err)
			continue
		}

		rr2, err := dns.NewRR(tt.String)
		if err != nil {
			t.Errorf("failed to convert example to dns.RR: %s", err)
			continue
		}

		if rr1.String() != rr2.String() {
			t.Errorf("unexpected RR:\nexpected: %s\nbut got:  %s", rr2, rr1)
		}
	}
}

func ExampleDomain() {
	a := landns.Domain("example.com")
	b := a.Normalized()
	fmt.Println(string(a), "->", string(b))

	c := landns.Domain("")
	d := c.Normalized()
	fmt.Println(string(c), "->", string(d))

	// Output:
	// example.com -> example.com.
	//  -> .
}

func ExampleNewRecord() {
	record, _ := landns.NewRecord("example.com. 600 IN A 127.0.0.1")

	fmt.Println(record.GetName())
	fmt.Println(record.GetTTL())
	fmt.Println(record.String())

	// Output:
	// example.com.
	// 600
	// example.com. 600 IN A 127.0.0.1
}

func ExampleNewRecordWithExpire() {
	record, _ := landns.NewRecordWithExpire("example.com. 600 IN A 127.0.0.1", time.Now().Add(10*time.Second))

	fmt.Println(record.GetName())
	fmt.Println(record.GetTTL())
	fmt.Println(record.String())

	// Output:
	// example.com.
	// 10
	// example.com. 10 IN A 127.0.0.1
}

func ExampleNewRecordFromRR() {
	rr, _ := dns.NewRR("example.com. 600 IN A 127.0.0.1")
	record, _ := landns.NewRecordFromRR(rr)

	fmt.Println(record.GetName())
	fmt.Println(record.GetTTL())
	fmt.Println(record.String())

	// Output:
	// example.com.
	// 600
	// example.com. 600 IN A 127.0.0.1
}
