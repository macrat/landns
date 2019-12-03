package landns_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns"
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
