package landns_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/client"
	"github.com/miekg/dns"
)

func StartServer(ctx context.Context, t testing.TB) (client.Client, *net.UDPAddr) {
	metrics := landns.NewMetrics("landns")
	dyn, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %s", err)
	}

	s := &landns.Server{
		Metrics:         metrics,
		DynamicResolver: dyn,
		Resolvers:       dyn,
	}

	apiAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9353}
	dnsAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3553}
	go func() {
		if err := s.ListenAndServe(ctx, apiAddr, dnsAddr, "udp"); err != nil {
			t.Fatalf("failed to start server: %s", err)
		}
	}()

	u, err := url.Parse(fmt.Sprintf("http://%s/api/v1/", apiAddr))
	if err != nil {
		t.Fatalf("failed to parse URL: %s", err)
	}

	time.Sleep(10 * time.Millisecond) // wait for start server

	return client.New(u), dnsAddr
}

func TestServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, d := StartServer(ctx, t)

	if rs, err := landns.NewDynamicRecordSet("example.com. 300 IN A 127.0.1.2"); err != nil {
		t.Errorf("failed to parse record set: %s", err)
	} else if err := c.Set(rs); err != nil {
		t.Errorf("failed to set dynamic record: %s", err)
	}

	if rs, err := c.Get(); err != nil {
		t.Errorf("failed to get records: %s", err)
	} else if len(rs) != 2 {
		t.Errorf("unexpected records:\n%s", rs)
	}

	AssertExchange(t, d, []dns.Question{
		{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}, "example.com.\t300\tIN\tA\t127.0.1.2")

	if resp, err := http.Get("http://127.0.0.1:9353"); err != nil {
		t.Errorf("failed to get index page: %s", err)
	} else {
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		expect := `<h1>Landns</h1><a href="/metrics">metrics</a> <a href="/api/v1">records</a>` + "\n"
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("failed to read index page: %s", err)
		} else if string(body) != expect {
			t.Errorf("unexpected index page:\nexpected:\n%s\nbut got:\n%s\n", expect, string(body))
		}
	}

	if resp, err := http.Get("http://127.0.0.1:9353/metrics"); err != nil {
		t.Errorf("failed to get metrics: %s", err)
	} else if resp.StatusCode != 200 {
		t.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestServer_StartStop(t *testing.T) {
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		StartServer(ctx, t)

		cancel()

		time.Sleep(10 * time.Millisecond) // wait for stop server
	}
}
