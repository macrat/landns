package landns_test

import (
	"fmt"
	"testing"

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
