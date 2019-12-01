package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

type FatalFormatter interface {
	Fatalf(fmt string, params ...interface{})
}

func createSqliteResolver(t FatalFormatter) *landns.SqliteResolver {
	metrics := landns.NewMetrics("landns")
	resolver, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %s", err.Error())
	}

	return resolver
}

func TestSqliteResolver_Addresses(t *testing.T) {
	resolver := createSqliteResolver(t)

	ttlA := uint32(123)
	ttlB := uint32(321)

	confA := landns.AddressesConfig{
		"example.com.": {
			{TTL: &ttlA, Address: net.ParseIP("127.0.0.1")},
			{TTL: &ttlB, Address: net.ParseIP("127.0.0.2")},
			{TTL: &ttlA, Address: net.ParseIP("1:1::1")},
		},
		"blanktar.jp.": {
			{TTL: &ttlA, Address: net.ParseIP("127.0.1.1")},
			{TTL: &ttlB, Address: net.ParseIP("1:2::1")},
		},
	}
	confB := landns.AddressesConfig{
		"blanktar.jp.": {
			{TTL: &ttlA, Address: net.ParseIP("1:2::2")},
		},
		"test.local.": {
			{TTL: &ttlB, Address: net.ParseIP("127.0.3.1")},
		},
	}
	expect := landns.AddressesConfig{
		"example.com.": {
			{TTL: &ttlA, Address: net.ParseIP("127.0.0.1")},
			{TTL: &ttlB, Address: net.ParseIP("127.0.0.2")},
			{TTL: &ttlA, Address: net.ParseIP("1:1::1")},
		},
		"blanktar.jp.": {
			{TTL: &ttlA, Address: net.ParseIP("1:2::2")},
		},
		"test.local.": {
			{TTL: &ttlB, Address: net.ParseIP("127.0.3.1")},
		},
	}

	resolver.UpdateAddresses(confA)
	resolver.UpdateAddresses(confB)

	all, err := resolver.GetAddresses()
	if err != nil {
		t.Errorf("failed to get addresses: %s", err.Error())
	}
	if err := all.Validate(); err != nil {
		t.Fatalf("invalid addresses: %s", err.Error())
	}
	if len(all) != 3 {
		t.Errorf("unexpected response length: expected 3 but got %d", len(all))
	}

	for name, records := range all {
		if rs, ok := expect[name]; !ok {
			t.Errorf("not found expected name: %s", name)
		} else if len(records) != len(rs) {
			t.Errorf("unexpected record length: %s: expected %d but got %d", name, len(rs), len(records))
		} else {
			for i := range records {
				if *records[i].TTL != *rs[i].TTL {
					t.Errorf("unexpected record: %s: expected %d but got %d", name, *rs[i].TTL, *records[i].TTL)
				}
			}
		}
	}

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeA, false),
		true,
		"example.com. 123 IN A 127.0.0.1",
		"example.com. 321 IN A 127.0.0.2",
	)

	ResolverTest(t, resolver, landns.NewRequest("1.0.0.127.in-addr.arpa.", dns.TypePTR, false), true, "1.0.0.127.in-addr.arpa. 123 IN PTR example.com.")
	ResolverTest(t, resolver, landns.NewRequest("2.0.0.127.in-addr.arpa.", dns.TypePTR, false), true, "2.0.0.127.in-addr.arpa. 321 IN PTR example.com.")

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeAAAA, false),
		true,
		"example.com. 123 IN AAAA 1:1::1",
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.1.0.0.0.ip6.arpa.", dns.TypePTR, false),
		true,
		"1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.1.0.0.0.ip6.arpa. 123 IN PTR example.com.",
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("blanktar.jp.", dns.TypeA, false),
		true,
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("blanktar.jp.", dns.TypeAAAA, false),
		true,
		"blanktar.jp. 123 IN AAAA 1:2::2",
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.0.1.0.0.0.ip6.arpa.", dns.TypePTR, false),
		true,
		"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.0.1.0.0.0.ip6.arpa. 123 IN PTR blanktar.jp.",
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("test.local.", dns.TypeA, false),
		true,
		"test.local. 321 IN A 127.0.3.1",
	)

	ResolverTest(t, resolver, landns.NewRequest("1.3.0.127.in-addr.arpa.", dns.TypePTR, false), true, "1.3.0.127.in-addr.arpa. 321 IN PTR test.local.")

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("test.local.", dns.TypeAAAA, false),
		true,
	)
}

func TestSqliteResolver_Cnames(t *testing.T) {
	resolver := createSqliteResolver(t)

	ttlA := uint32(123)
	ttlB := uint32(321)

	confA := landns.CnamesConfig{
		"example.com.": {
			{TTL: &ttlA, Target: "a.example.com."},
			{TTL: &ttlB, Target: "b.example.com."},
		},
		"blanktar.jp.": {
			{TTL: &ttlA, Target: "c.example.com."},
			{TTL: &ttlB, Target: "d.example.com."},
		},
	}
	confB := landns.CnamesConfig{
		"blanktar.jp.": {
			{TTL: &ttlA, Target: "e.example.com."},
		},
		"test.local.": {
			{TTL: &ttlB, Target: "f.example.com."},
		},
	}
	expect := landns.CnamesConfig{
		"example.com.": {
			{TTL: &ttlA, Target: "a.example.com."},
			{TTL: &ttlB, Target: "b.example.com."},
		},
		"blanktar.jp.": {
			{TTL: &ttlA, Target: "e.example.com."},
		},
		"test.local.": {
			{TTL: &ttlB, Target: "f.example.com."},
		},
	}

	resolver.UpdateCnames(confA)
	resolver.UpdateCnames(confB)

	all, err := resolver.GetCnames()
	if err != nil {
		t.Errorf("failed to get cnames: %s", err.Error())
	}
	if err := all.Validate(); err != nil {
		t.Fatalf("invalid cnames: %s", err.Error())
	}
	if len(all) != 3 {
		t.Errorf("unexpected response length: expected 3 but got %d", len(all))
	}

	for name, records := range all {
		if rs, ok := expect[name]; !ok {
			t.Errorf("not found expected name: %s", name)
		} else if len(records) != len(rs) {
			t.Errorf("unexpected record length: %s: expected %d but got %d", name, len(rs), len(records))
		} else {
			for i := range records {
				if *records[i].TTL != *rs[i].TTL {
					t.Errorf("unexpected record: %s: expected %d but got %d", name, *rs[i].TTL, *records[i].TTL)
				}
			}
		}
	}

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeCNAME, false),
		true,
		"example.com. 123 IN CNAME a.example.com.",
		"example.com. 321 IN CNAME b.example.com.",
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("blanktar.jp.", dns.TypeCNAME, false),
		true,
		"blanktar.jp. 123 IN CNAME e.example.com.",
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("test.local.", dns.TypeCNAME, false),
		true,
		"test.local. 321 IN CNAME f.example.com.",
	)
}

func TestSqliteResolver_Texts(t *testing.T) {
	resolver := createSqliteResolver(t)

	ttlA := uint32(123)
	ttlB := uint32(321)

	confA := landns.TextsConfig{
		"example.com.": {
			{TTL: &ttlA, Text: "abc"},
			{TTL: &ttlB, Text: "def"},
		},
		"blanktar.jp.": {
			{TTL: &ttlA, Text: "ghi"},
			{TTL: &ttlB, Text: "jkl"},
		},
	}
	confB := landns.TextsConfig{
		"blanktar.jp.": {
			{TTL: &ttlA, Text: "mno"},
		},
		"test.local.": {
			{TTL: &ttlB, Text: "pqr"},
		},
	}
	expect := landns.TextsConfig{
		"example.com.": {
			{TTL: &ttlA, Text: "abc"},
			{TTL: &ttlB, Text: "def"},
		},
		"blanktar.jp.": {
			{TTL: &ttlA, Text: "mno"},
		},
		"test.local.": {
			{TTL: &ttlB, Text: "pqr"},
		},
	}

	resolver.UpdateTexts(confA)
	resolver.UpdateTexts(confB)

	all, err := resolver.GetTexts()
	if err != nil {
		t.Errorf("failed to get texts: %s", err.Error())
	}
	if err := all.Validate(); err != nil {
		t.Fatalf("invalid texts: %s", err.Error())
	}
	if len(all) != 3 {
		t.Errorf("unexpected response length: expected 3 but got %d", len(all))
	}

	for name, records := range all {
		if rs, ok := expect[name]; !ok {
			t.Errorf("not found expected name: %s", name)
		} else if len(records) != len(rs) {
			t.Errorf("unexpected record length: %s: expected %d but got %d", name, len(rs), len(records))
		} else {
			for i := range records {
				if *records[i].TTL != *rs[i].TTL {
					t.Errorf("unexpected record: %s: expected %d but got %d", name, *rs[i].TTL, *records[i].TTL)
				}
			}
		}
	}

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("example.com.", dns.TypeTXT, false),
		true,
		`example.com. 123 IN TXT "abc"`,
		`example.com. 321 IN TXT "def"`,
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("blanktar.jp.", dns.TypeTXT, false),
		true,
		`blanktar.jp. 123 IN TXT "mno"`,
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("test.local.", dns.TypeTXT, false),
		true,
		`test.local. 321 IN TXT "pqr"`,
	)
}

func TestSqliteResolver_Services(t *testing.T) {
	resolver := createSqliteResolver(t)

	ttlA := uint32(123)
	ttlB := uint32(321)

	confA := landns.ServicesConfig{
		"_web._tcp.example.com.": {
			{TTL: &ttlA, Priority: 1, Weight: 2, Port: 3, Target: "a.example.com."},
			{TTL: &ttlB, Priority: 4, Weight: 5, Port: 6, Target: "b.example.com."},
		},
		"_ftp._tcp.blanktar.jp.": {
			{TTL: &ttlA, Priority: 7, Weight: 8, Port: 9, Target: "c.example.com."},
			{TTL: &ttlB, Priority: 10, Weight: 11, Port: 12, Target: "d.example.com."},
		},
	}
	confB := landns.ServicesConfig{
		"_ftp._tcp.blanktar.jp.": {
			{TTL: &ttlA, Priority: 13, Weight: 14, Port: 15, Target: "e.example.com."},
		},
		"_dns._udp.test.local.": {
			{TTL: &ttlA, Priority: 16, Weight: 17, Port: 18, Target: "f.example.com."},
		},
	}
	expect := landns.ServicesConfig{
		"_web._tcp.example.com.": {
			{TTL: &ttlA, Priority: 1, Weight: 2, Port: 3, Target: "a.example.com."},
			{TTL: &ttlB, Priority: 4, Weight: 5, Port: 6, Target: "b.example.com."},
		},
		"_ftp._tcp.blanktar.jp.": {
			{TTL: &ttlA, Priority: 13, Weight: 14, Port: 15, Target: "e.example.com."},
		},
		"_dns._udp.test.local.": {
			{TTL: &ttlA, Priority: 16, Weight: 17, Port: 18, Target: "f.example.com."},
		},
	}

	resolver.UpdateServices(confA)
	resolver.UpdateServices(confB)

	all, err := resolver.GetServices()
	if err != nil {
		t.Errorf("failed to get services: %s", err.Error())
	}
	if err := all.Validate(); err != nil {
		t.Fatalf("invalid services: %s", err.Error())
	}
	if len(all) != 3 {
		t.Errorf("unexpected response length: expected 3 but got %d", len(all))
	}

	for name, records := range all {
		if rs, ok := expect[name]; !ok {
			t.Errorf("not found expected name: %s", name)
		} else if len(records) != len(rs) {
			t.Errorf("unexpected record length: %s: expected %d but got %d", name, len(rs), len(records))
		} else {
			for i := range records {
				if *records[i].TTL != *rs[i].TTL {
					t.Errorf("unexpected record: %s: expected %d but got %d", name, *rs[i].TTL, *records[i].TTL)
				}
			}
		}
	}

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("_web._tcp.example.com.", dns.TypeSRV, false),
		true,
		`_web._tcp.example.com. 123 IN SRV 1 2 3 a.example.com.`,
		`_web._tcp.example.com. 321 IN SRV 4 5 6 b.example.com.`,
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("_ftp._tcp.blanktar.jp.", dns.TypeSRV, false),
		true,
		`_ftp._tcp.blanktar.jp. 123 IN SRV 13 14 15 e.example.com.`,
	)

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("_dns._udp.test.local.", dns.TypeSRV, false),
		true,
		`_dns._udp.test.local. 123 IN SRV 16 17 18 f.example.com.`,
	)
}

func BenchmarkSqliteResolver(b *testing.B) {
	resolver := createSqliteResolver(b)

	config := landns.AddressesConfig{}

	for i := 0; i < 100; i++ {
		config[landns.Domain(fmt.Sprintf("host%d.example.com.", i))] = []landns.AddressRecordConfig{
			{Address: net.ParseIP("127.1.2.3")},
			{Address: net.ParseIP("127.1.2.4")},
		}
	}

	resolver.UpdateAddresses(config)

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(NewDummyResponseWriter(), req)
	}
}
