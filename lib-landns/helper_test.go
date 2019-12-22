package landns_test

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func FindEmptyPort() int {
	for port := 49152; port <= 65535; port++ {
		l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			l.Close()
			return port
		}
	}
	return -1
}

func AssertResolve(t testing.TB, resolver landns.Resolver, request landns.Request, authoritative bool, responses ...string) {
	t.Helper()

	resp := NewDummyResponseWriter()
	if err := resolver.Resolve(resp, request); err != nil {
		t.Errorf("%s <- %s: failed to resolve: %v", resolver, request, err.Error())
		return
	}

	if resp.Authoritative != authoritative {
		t.Errorf("%s <- %s: unexpected authoritive of response: expected %v but got %v", resolver, request, authoritative, resp.Authoritative)
	}

	sort.Slice(resp.Records, func(i, j int) bool {
		return strings.Compare(resp.Records[i].String(), resp.Records[j].String()) == 1
	})
	sort.Slice(responses, func(i, j int) bool {
		return strings.Compare(responses[i], responses[j]) == 1
	})

	ok := len(resp.Records) == len(responses)

	if ok {
		for i := range responses {
			if resp.Records[i].String() != responses[i] {
				ok = false
				break
			}
		}
	}

	if !ok {
		msg := fmt.Sprintf("%s <- %s: unexpected resolve response:\nexpected:\n", resolver, request)
		for _, x := range responses {
			msg += "\t" + x + "\n"
		}
		msg += "but got:\n"
		for _, x := range resp.Records {
			msg += "\t" + x.String() + "\n"
		}
		t.Errorf("%s", msg)
	}
}

func AssertExchange(t *testing.T, addr *net.UDPAddr, question []dns.Question, expect ...string) {
	t.Helper()

	msg := &dns.Msg{
		MsgHdr:   dns.MsgHdr{Id: dns.Id()},
		Question: question,
	}

	in, err := dns.Exchange(msg, addr.String())
	if err != nil {
		t.Errorf("%s: failed to resolve: %s", addr, err)
		return
	}

	ok := len(in.Answer) == len(expect)

	if ok {
		for i := range expect {
			if in.Answer[i].String() != expect[i] {
				ok = false
				break
			}
		}
	}

	if !ok {
		msg := "%s: unexpected answer:\nexpected:\n"
		for _, x := range expect {
			msg += x + "\n"
		}
		msg += "\nbut got:\n"
		for _, x := range in.Answer {
			msg += x.String() + "\n"
		}
		t.Errorf(msg, addr)
	}
}

func CheckRecursionAvailable(t testing.TB, makeResolver func([]landns.Resolver) landns.Resolver) {
	t.Helper()

	recursionResolver := DummyResolver{false, true}
	nonRecursionResolver := DummyResolver{false, false}

	resolver := makeResolver([]landns.Resolver{nonRecursionResolver, recursionResolver, nonRecursionResolver})
	if resolver.RecursionAvailable() != true {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}

	resolver = makeResolver([]landns.Resolver{nonRecursionResolver, nonRecursionResolver})
	if resolver.RecursionAvailable() != false {
		t.Fatalf("unexpected recursion available: %v", recursionResolver.RecursionAvailable())
	}
}
