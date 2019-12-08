package testutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func StartDummyDNSServer(ctx context.Context, t FatalFormatter, resolver landns.Resolver) *net.UDPAddr {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3553}

	server := dns.Server{
		Addr:      addr.String(),
		Net:       "udp",
		ReusePort: true,
		Handler:   landns.NewHandler(resolver, landns.NewMetrics("landns")),
	}

	go func() {
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			t.Fatalf("failed to serve dummy DNS: %s", err)
		}
	}()

	go func() {
		<-ctx.Done()
		server.Shutdown()
	}()

	time.Sleep(10 * time.Millisecond) // Wait for start DNS server

	return addr
}

func StartDummyMetricsServer(ctx context.Context, t FatalFormatter, namespace string) (*landns.Metrics, func() string) {
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5380}

	metrics := landns.NewMetrics("landns")
	handler, err := metrics.HTTPHandler()
	if err != nil {
		t.Fatalf("failed to serve dummy metrics server: %s", err)
	}

	server := http.Server{
		Addr:    addr.String(),
		Handler: handler,
	}

	go func() {
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			t.Fatalf("failed to serve dummy metrics server: %s", err)
		}
	}()

	go func() {
		<-ctx.Done()
		c, _ := context.WithTimeout(context.Background(), 1*time.Second)
		if err := server.Shutdown(c); err != nil {
			t.Fatalf("failed to stop dummy metrics server: %s", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // Wait for start DNS server

	return metrics, func() string {
		resp, err := http.Get(fmt.Sprintf("http://%s", addr))
		if err != nil {
			t.Fatalf("failed to get metrics: %s", err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to get metrics: %s", err)
		}

		return string(body)
	}
}
