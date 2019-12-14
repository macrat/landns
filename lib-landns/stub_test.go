package landns_test

import (
	"fmt"
	"net"
	"testing"
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

type DummyResolver struct {
	Error     bool
	Recursion bool
}

func (dr DummyResolver) Resolve(w landns.ResponseWriter, r landns.Request) error {
	if dr.Error {
		return fmt.Errorf("test error")
	} else {
		return nil
	}
}

func (dr DummyResolver) RecursionAvailable() bool {
	return dr.Recursion
}

func (dr DummyResolver) Close() error {
	return nil
}

func TestDummyResolver(t *testing.T) {
	tests := []struct{
		err bool
		rec bool
	}{
		{false, false},
		{false, true},
		{true, false},
		{true, true},
	}

	w := NewDummyResponseWriter()
	r := landns.NewRequest("example.com.", dns.TypeA, true)

	for _, tt := range tests {
		res := DummyResolver{tt.err, tt.rec}

		if err := res.Resolve(w, r); !tt.err && err != nil {
			t.Errorf("unexpected error: %#v", err)
		} else if tt.err && err == nil {
			t.Errorf("expected error but not occurred")
		}

		if res.RecursionAvailable() != tt.rec {
			t.Errorf("unexpected recursion available: expected %#v but got %#v", tt.rec, res.RecursionAvailable())
		}

		if err := res.Close(); err != nil {
			t.Errorf("unexpected error: %#v", err)
		}
	}
}

type DummyResponseWriter struct {
	Records       []landns.Record
	Authoritative bool
}

func NewDummyResponseWriter() *DummyResponseWriter {
	return &DummyResponseWriter{
		Records:       make([]landns.Record, 0, 10),
		Authoritative: true,
	}
}

func (rw *DummyResponseWriter) Add(r landns.Record) error {
	rw.Records = append(rw.Records, r)
	return nil
}

func (rw *DummyResponseWriter) IsAuthoritative() bool {
	return rw.Authoritative
}

func (rw *DummyResponseWriter) SetNoAuthoritative() {
	rw.Authoritative = false
}

type EmptyResponseWriter struct{}

func (rw EmptyResponseWriter) Add(r landns.Record) error {
	return nil
}

func (rw EmptyResponseWriter) IsAuthoritative() bool {
	return true
}

func (rw EmptyResponseWriter) SetNoAuthoritative() {
}

func TestDummyResponseWriter(t *testing.T) {
	w := NewDummyResponseWriter()

	if w.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: expected true but got false")
	}
	w.SetNoAuthoritative()
	if w.IsAuthoritative() != false {
		t.Errorf("unexpected authoritative: expected false but got true")
	}

	records := []landns.Record{
		landns.AddressRecord{"example.com.", 42, net.ParseIP("127.1.2.3")},
		landns.AddressRecord{"example.com.", 123, net.ParseIP("127.9.8.7")},
	}
	for _, r := range records {
		if err := w.Add(r); err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
	}

	for i := range w.Records {
		if records[i].String() != w.Records[i].String() {
			t.Errorf("unexpected record: expected %#v but got %#v", records[i], w.Records[i])
		}
	}
}

func TestEmptyResponseWriter(t *testing.T) {
	w := EmptyResponseWriter{}

	if w.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: expected true but got false")
	}
	w.SetNoAuthoritative()
	if w.IsAuthoritative() != true {
		t.Errorf("unexpected authoritative: expected true but got false")
	}

	if err := w.Add(landns.AddressRecord{"example.com.", 42, net.ParseIP("127.1.2.3")}); err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func StartDummyDNSServer(ctx context.Context, t testing.TB, resolver landns.Resolver) *net.UDPAddr {
	t.Helper()

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

func StartDummyMetricsServer(ctx context.Context, t testing.TB, namespace string) (*landns.Metrics, func() string) {
	t.Helper()

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
