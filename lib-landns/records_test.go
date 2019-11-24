package landns_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns"
)

func TestDomainValidate(t *testing.T) {
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
}

func TestProtoValidate(t *testing.T) {
	a := landns.Proto("")
	if err := a.Validate(); err != nil {
		t.Errorf("failed to empty proto validation: %#v", err.Error())
	}

	b := landns.Proto("foobar")
	if err := b.Validate(); err == nil {
		t.Errorf("failed to invalid proto validation: <nil>")
	} else if err.Error() != `invalid proto: "foobar"` {
		t.Errorf("failed to invalid proto validation: %#v", err.Error())
	}

	c := landns.Proto("tcp")
	if err := c.Validate(); err != nil {
		t.Errorf("failed to tcp proto validation: %#v", err.Error())
	}

	d := landns.Proto("udp")
	if err := d.Validate(); err != nil {
		t.Errorf("failed to udp proto validation: %#v", err.Error())
	}
}
