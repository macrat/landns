package landns_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

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

type EmptyResponseWriter struct {}

func (rw EmptyResponseWriter) Add(r landns.Record) error {
	return nil
}

func (rw EmptyResponseWriter) IsAuthoritative() bool {
	return true
}

func (rw EmptyResponseWriter) SetNoAuthoritative() {
}

func ResolverTest(t *testing.T, resolver landns.Resolver, request landns.Request, authoritative bool, responses ...string) {
	resp := NewDummyResponseWriter()
	if err := resolver.Resolve(resp, request); err != nil {
		t.Errorf("%s <- %s: failed to resolve: %v", resolver, request, err.Error())
		return
	}

	if resp.Authoritative != authoritative {
		t.Errorf(`%s <- %s: unexcepted authoritive of response: excepted %v but got %v`, resolver, request, authoritative, resp.Authoritative)
	}

	if len(resp.Records) != len(responses) {
		t.Errorf(`%s <- %s: unexcepted resolve response: excepted length %d but got %d`, resolver, request, len(responses), len(resp.Records))
		return
	}

	for i, _ := range responses {
		if resp.Records[i].String() != responses[i] {
			t.Errorf(`%s <- %s: unexcepted resolve response: excepted "%s" but got "%s"`, resolver, request, responses[i], resp.Records[i])
		}
	}
}

func BenchmarkResolverSet(b *testing.B) {
	resolver := make(landns.ResolverSet, 100)

	for i := 0; i < 100; i++ {
		host := fmt.Sprintf("host%d.example.com.", i)

		resolver[i] = landns.NewSimpleResolver([]landns.Record{
			landns.AddressRecord{
				Name:    landns.Domain(host),
				Address: net.ParseIP(fmt.Sprintf("127.0.0.%d", i)),
			},
		})
	}

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)
	resp := EmptyResponseWriter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(resp, req)
	}
}
