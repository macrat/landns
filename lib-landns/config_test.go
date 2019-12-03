package landns_test

import (
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
)

func TestProto_Validate(t *testing.T) {
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

func TestProto_Encoding(t *testing.T) {
	var p landns.Proto

	for input, expect := range map[string]string{"tcp": "tcp", "udp": "udp", "": "tcp"} {
		if err := (&p).UnmarshalText([]byte(input)); err != nil {
			t.Errorf("failed to unmarshal: %s: %s", input, err)
		} else if result, err := p.MarshalText(); err != nil {
			t.Errorf("failed to marshal: %s: %s", input, err)
		} else if string(result) != expect {
			t.Errorf("unexpected marshal result: expected %s but got %s", expect, string(result))
		}
	}

	if err := (&p).UnmarshalText([]byte("foo")); err == nil {
		t.Errorf("expected error but got nil")
	} else if err.Error() != `invalid proto: "foo"` {
		t.Errorf(`unexpected error: expected 'invalid proto: "foo"' but got '%s'`, err)
	}
}

func TestAddressRecordConfigNormalized(t *testing.T) {
	a := landns.AddressRecordConfig{nil, net.ParseIP("127.0.1.2")}
	an := a.Normalized()

	if string(a.Address) != string(an.Address) {
		t.Errorf("failed to copy address: %s != %s", a.Address, an.Address)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v", an.TTL)
	} else if *an.TTL != landns.DefaultTTL {
		t.Errorf("failed to set DefaultTTL: %v", *an.TTL)
	}

	ttl := uint32(42)
	b := landns.AddressRecordConfig{&ttl, net.ParseIP("127.3.2.1")}
	bn := b.Normalized()

	if string(b.Address) != string(bn.Address) {
		t.Errorf("failed to copy address: %s != %s", b.Address, bn.Address)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DefaultTTL: %v != %v", *b.TTL, *bn.TTL)
	}
}

func TestCnameRecordConfigNormalized(t *testing.T) {
	a := landns.CnameRecordConfig{nil, landns.Domain("example.com")}
	an := a.Normalized()

	if string(an.Target) != "example.com." {
		t.Errorf("failed to normalize target domain: %v", an.Target)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v", an.TTL)
	} else if *an.TTL != landns.DefaultTTL {
		t.Errorf("failed to set DefaultTTL: %v", *an.TTL)
	}

	ttl := uint32(42)
	b := landns.CnameRecordConfig{&ttl, landns.Domain("foo.bar.")}
	bn := b.Normalized()

	if string(b.Target) != string(bn.Target) {
		t.Errorf("failed to copy target domain name: %s != %s", b.Target, bn.Target)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DefaultTTL: %v != %v", *b.TTL, *bn.TTL)
	}
}

func TestTxtRecordConfigNormalized(t *testing.T) {
	a := landns.TxtRecordConfig{nil, "hello_world"}
	an := a.Normalized()

	if an.Text != a.Text {
		t.Errorf("failed to copy text: %v", an.Text)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v", an.TTL)
	} else if *an.TTL != landns.DefaultTTL {
		t.Errorf("failed to set DefaultTTL: %v", *an.TTL)
	}

	ttl := uint32(42)
	b := landns.TxtRecordConfig{&ttl, "foo_bar"}
	bn := b.Normalized()

	if b.Text != bn.Text {
		t.Errorf("failed to copy text: %s != %s", b.Text, bn.Text)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DefaultTTL: %v != %v", *b.TTL, *bn.TTL)
	}
}

func TestSrvRecordConfigNormalized(t *testing.T) {
	a := landns.SrvRecordConfig{TTL: nil, Target: landns.Domain("example.com")}
	an := a.Normalized()

	if string(an.Target) != "example.com." {
		t.Errorf("failed to normalize target domain: %v", an.Target)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v", an.TTL)
	} else if *an.TTL != landns.DefaultTTL {
		t.Errorf("failed to set DefaultTTL: %v", *an.TTL)
	}

	ttl := uint32(42)
	b := landns.SrvRecordConfig{TTL: &ttl, Target: landns.Domain("foo.bar.")}
	bn := b.Normalized()

	if string(b.Target) != string(bn.Target) {
		t.Errorf("failed to copy: %v != %v", b.Target, bn.Target)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DefaultTTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DefaultTTL: %v != %v", *b.TTL, *bn.TTL)
	}
}
