package landns_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func StartDummyMetricsServer(ctx context.Context, t *testing.T, namespace string) (*landns.Metrics, func() string) {
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
			t.Errorf("failed to stop dummy metrics server: %s", err)
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

func MetricsResponseTest(t *testing.T, name, metrics string, re *regexp.Regexp, expect int) {
	result := re.FindStringSubmatch(metrics)

	if len(result) != 2 {
		t.Errorf("unexpected %s length: expected 2 but got %d", name, len(result))
	} else if result[1] != fmt.Sprint(expect) {
		t.Errorf("unexpected %s value: expected %d but got %s", name, expect, result[1])
	}
}

func TestMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics, get := StartDummyMetricsServer(ctx, t, "landns")

	for i, test := range []struct {
		Name           string
		Re             *regexp.Regexp
		Authoritative  bool
		ResponseLength int
	}{
		{"local resolve count", regexp.MustCompile(`landns_resolve_count\{.*source="local".*type="A".*\} (.*)`), true, 1},
		{"forward resolve count", regexp.MustCompile(`landns_resolve_count\{.*source="upstream".*type="A".*\} (.*)`), false, 1},
		{"resolve missing count", regexp.MustCompile(`landns_resolve_count\{.*source="not-found".*type="A".*\} (.*)`), true, 0},
	} {
		MetricsResponseTest(t, test.Name, get(), test.Re, 0)
		MetricsResponseTest(t, "message count", get(), regexp.MustCompile(`landns_received_message_count\{.*type="query"\} (.*)`), i)

		req := &dns.Msg{
			MsgHdr: dns.MsgHdr{Id: dns.Id()},
			Question: []dns.Question{
				{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			},
		}
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Authoritative = test.Authoritative
		resp.Answer = []dns.RR{}
		for i := 0; i < test.ResponseLength; i++ {
			rr, err := dns.NewRR(fmt.Sprintf("example.com. 42 IN A 127.0.0.%d", i))
			if err != nil {
				t.Fatalf("failed to make RR: %s", err)
			}
			resp.Answer = append(resp.Answer, rr)
		}

		metrics.Start(req)(resp)

		MetricsResponseTest(t, test.Name, get(), test.Re, 1)
		MetricsResponseTest(t, "message count", get(), regexp.MustCompile(`landns_received_message_count\{.*type="query".*\} (.*)`), i+1)
	}

	MetricsResponseTest(t, "skip count", get(), regexp.MustCompile(`landns_received_message_count\{.*type="another".*\} (.*)`), 0)
	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id(), Opcode: dns.OpcodeNotify},
	}
	resp := new(dns.Msg)
	resp.SetReply(req)
	metrics.Start(req)(resp)
	MetricsResponseTest(t, "skip count", get(), regexp.MustCompile(`landns_received_message_count\{.*type="another".*\} (.*)`), 1)

	MetricsResponseTest(t, "error count", get(), regexp.MustCompile(`landns_resolve_error_count\{.*\} (.*)`), 0)
	metrics.Error(landns.NewRequest("example.com.", dns.TypeA, true), fmt.Errorf("test error"))
	MetricsResponseTest(t, "error count", get(), regexp.MustCompile(`landns_resolve_error_count\{.*\} (.*)`), 1)
}

func BenchmarkMetrics(b *testing.B) {
	metrics := landns.NewMetrics("landns")

	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: dns.Id()},
		Question: []dns.Question{
			{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		},
	}
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	rr, err := dns.NewRR("example.com. 42 IN A 127.0.0.1")
	if err != nil {
		b.Fatalf("failed to make RR: %s", err)
	}
	resp.Answer = []dns.RR{rr}

	b.ResetTimer()

	for i := 0; i<b.N; i++ {
		metrics.Start(req)(resp)
	}
}
