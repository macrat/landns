package landns_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func TestStaticResolver(t *testing.T) {
	config := []byte(`ttl: 128

address:
  example.com: [127.1.2.3]
  server.example.com.:
    - 192.168.1.2
    - 192.168.1.3
    - 1:2::3

cname:
  file.example.com: [server.example.com.]

text:
  example.com:
    - hello world
    - foo

service:
  example.com:
    - service: ftp
      proto: tcp
      priority: 1
      weight: 2
      port: 21
      target: file.example.com
    - service: http
      port: 80
      target: server.example.com
`)

	resolver, err := landns.NewStaticResolver(config)
	if err != nil {
		t.Fatalf("failed to parse config: %s", err.Error())
	}

	for _, subResolver := range resolver {
		if r, ok := subResolver.(landns.ValidatableResolver); !ok {
			t.Errorf("unexcepted type of sub resolver: %#v", subResolver)
		} else if err := r.Validate(); err != nil {
			t.Fatalf("invalid resolver state: %s: %s", r, err)
		}
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 128 A 127.1.2.3")
	ResolverTest(t, resolver, landns.NewRequest("server.example.com.", dns.TypeA, false), true, "server.example.com. 128 A 192.168.1.2", "server.example.com. 128 A 192.168.1.3")

	ResolverTest(t, resolver, landns.NewRequest("server.example.com.", dns.TypeAAAA, false), true, "server.example.com. 128 AAAA 1:2::3")

	ResolverTest(t, resolver, landns.NewRequest("3.2.1.127.in-addr.arpa.", dns.TypePTR, false), true, "3.2.1.127.in-addr.arpa. 128 PTR example.com.")
	ResolverTest(t, resolver, landns.NewRequest("2.1.168.192.in-addr.arpa.", dns.TypePTR, false), true, "2.1.168.192.in-addr.arpa. 128 PTR server.example.com.")
	ResolverTest(t, resolver, landns.NewRequest("3.1.168.192.in-addr.arpa.", dns.TypePTR, false), true, "3.1.168.192.in-addr.arpa. 128 PTR server.example.com.")

	ResolverTest(
		t,
		resolver,
		landns.NewRequest("3.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.0.1.0.0.0.ip6.arpa.", dns.TypePTR, false),
		true,
		"3.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.0.1.0.0.0.ip6.arpa. 128 PTR server.example.com.",
	)

	ResolverTest(t, resolver, landns.NewRequest("file.example.com.", dns.TypeCNAME, false), true, "file.example.com. 128 CNAME server.example.com.")
	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeTXT, false), true, `example.com. 128 TXT "hello world"`, `example.com. 128 TXT "foo"`)

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeSRV, false), true, "_ftp._tcp.example.com. 128 IN SRV 1 2 21 file.example.com.", "_http._tcp.example.com. 128 IN SRV 0 0 80 server.example.com.")
}

func TestStaticResolver_WithoutTTL(t *testing.T) {
	config := []byte(`address: {example.com: [127.1.2.3]}`)

	resolver, err := landns.NewStaticResolver(config)
	if err != nil {
		t.Fatalf("failed to parse config: %s", err.Error())
	}

	ResolverTest(t, resolver, landns.NewRequest("example.com.", dns.TypeA, false), true, "example.com. 3600 A 127.1.2.3")
}
