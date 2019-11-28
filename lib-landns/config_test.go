package landns_test

import (
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
)

func TestAddressRecordConfigNormalized(t *testing.T) {
	a := landns.AddressRecordConfig{nil, net.ParseIP("127.0.1.2")}
	an := a.Normalized()

	if string(a.Address) != string(an.Address) {
		t.Errorf("failed to copy address: %s != %s", a.Address, an.Address)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v", an.TTL)
	} else if *an.TTL != landns.DEFAULT_TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v", *an.TTL)
	}

	p := uint16(42)
	b := landns.AddressRecordConfig{&p, net.ParseIP("127.3.2.1")}
	bn := b.Normalized()

	if string(b.Address) != string(bn.Address) {
		t.Errorf("failed to copy address: %s != %s", b.Address, bn.Address)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v != %v", *b.TTL, *bn.TTL)
	}
}

func TestCnameRecordConfigNormalized(t *testing.T) {
	a := landns.CnameRecordConfig{nil, landns.Domain("example.com")}
	an := a.Normalized()

	if string(an.Target) != "example.com." {
		t.Errorf("failed to normalize target domain: %v", an.Target)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v", an.TTL)
	} else if *an.TTL != landns.DEFAULT_TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v", *an.TTL)
	}

	p := uint16(42)
	b := landns.CnameRecordConfig{&p, landns.Domain("foo.bar.")}
	bn := b.Normalized()

	if string(b.Target) != string(bn.Target) {
		t.Errorf("failed to copy target domain name: %s != %s", b.Target, bn.Target)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v != %v", *b.TTL, *bn.TTL)
	}
}

func TestTxtRecordConfigNormalized(t *testing.T) {
	a := landns.TxtRecordConfig{nil, "hello_world"}
	an := a.Normalized()

	if an.Text != a.Text {
		t.Errorf("failed to copy text: %v", an.Text)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v", an.TTL)
	} else if *an.TTL != landns.DEFAULT_TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v", *an.TTL)
	}

	p := uint16(42)
	b := landns.TxtRecordConfig{&p, "foo_bar"}
	bn := b.Normalized()

	if b.Text != bn.Text {
		t.Errorf("failed to copy text: %s != %s", b.Text, bn.Text)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v != %v", *b.TTL, *bn.TTL)
	}
}

func TestSrvRecordConfigNormalized(t *testing.T) {
	a := landns.SrvRecordConfig{TTL: nil, Proto: landns.Proto(""), Target: landns.Domain("example.com")}
	an := a.Normalized()

	if string(an.Target) != "example.com." {
		t.Errorf("failed to normalize target domain: %v", an.Target)
	}
	if string(an.Proto) != "tcp" {
		t.Errorf("failed to normalize protocol: %v", an.Proto)
	}
	if an.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v", an.TTL)
	} else if *an.TTL != landns.DEFAULT_TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v", *an.TTL)
	}

	p := uint16(42)
	b := landns.SrvRecordConfig{TTL: &p, Proto: landns.Proto("udp"), Target: landns.Domain("foo.bar.")}
	bn := b.Normalized()

	if string(b.Target) != string(bn.Target) {
		t.Errorf("failed to copy: %v != %v", b.Target, bn.Target)
	}
	if string(bn.Proto) != "udp" {
		t.Errorf("failed to copy protocol: %v != tcp", bn.Proto)
	}
	if b.TTL == nil || bn.TTL == nil {
		t.Errorf("failed to set DEFAULT_TTL: %#v != %#v", b.TTL, bn.TTL)
	} else if *b.TTL != *bn.TTL {
		t.Errorf("failed to set DEFAULT_TTL: %v != %v", *b.TTL, *bn.TTL)
	}
}
