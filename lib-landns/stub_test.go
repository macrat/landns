package landns_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
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
	tests := []struct {
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

	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: FindEmptyPort()}

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

type HTTPEndpoint struct {
	Test testing.TB
	URL  *url.URL
}

func (e HTTPEndpoint) Do(method, path, body string) (int, string, error) {
	e.Test.Helper()

	u, err := e.URL.Parse(path)
	if err != nil {
		e.Test.Errorf("failed to %s %s: %s", method, path, err)
		return 0, "", err
	}

	req, err := http.NewRequest(method, u.String(), strings.NewReader(body))
	if err != nil {
		e.Test.Errorf("failed to %s %s: %s", method, path, err)
		return 0, "", err
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		e.Test.Errorf("failed to %s %s: %s", method, path, err)
		return 0, "", err
	}
	defer resp.Body.Close()

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		e.Test.Errorf("failed to %s %s: %s", method, path, err)
		return 0, "", err
	}

	return resp.StatusCode, string(rbody), nil
}

func (e HTTPEndpoint) Get(path string) (string, error) {
	e.Test.Helper()

	status, body, err := e.Do("GET", path, "")
	if status != 200 {
		e.Test.Errorf("failed to get %s: status code %d", path, status)
	}

	return body, err
}

func (e HTTPEndpoint) Post(path, body string) (string, error) {
	e.Test.Helper()

	status, rbody, err := e.Do("POST", path, body)
	if status != 200 {
		e.Test.Errorf("failed to get %s: status code %d", path, status)
	}

	return rbody, err
}

func StartHTTPServer(ctx context.Context, t testing.TB, handler http.Handler) HTTPEndpoint {
	t.Helper()

	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: FindEmptyPort()}

	u, err := url.Parse(fmt.Sprintf("http://%s", addr))
	if err != nil {
		t.Fatalf("failed to make URL: %s", err)
	}

	server := http.Server{
		Addr:    addr.String(),
		Handler: handler,
	}

	go func() {
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			t.Fatalf("failed to serve HTTP server: %s", err)
		}
	}()

	go func() {
		<-ctx.Done()
		c, _ := context.WithTimeout(context.Background(), 1*time.Second)
		if err := server.Shutdown(c); err != nil {
			t.Fatalf("failed to stop HTTP server: %s", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // Wait for start DNS server

	return HTTPEndpoint{URL: u, Test: t}
}

func StartDummyMetricsServer(ctx context.Context, t testing.TB, namespace string) (*landns.Metrics, func() (string, error)) {
	t.Helper()

	metrics := landns.NewMetrics("landns")
	handler, err := metrics.HTTPHandler()
	if err != nil {
		t.Fatalf("failed to serve dummy metrics server: %s", err)
	}

	u := StartHTTPServer(ctx, t, handler)

	return metrics, func() (string, error) {
		return u.Get("/")
	}
}
