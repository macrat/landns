package landns_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns"
)

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

	if len(resp.Records) != len(responses) {
		t.Errorf("%s <- %s: unexpected resolve response: expected length %d but got %d", resolver, request, len(responses), len(resp.Records))
		return
	}

	sort.Slice(resp.Records, func(i, j int) bool {
		return strings.Compare(resp.Records[i].String(), resp.Records[j].String()) == 1
	})
	sort.Slice(responses, func(i, j int) bool {
		return strings.Compare(responses[i], responses[j]) == 1
	})

	for i := range responses {
		if resp.Records[i].String() != responses[i] {
			t.Errorf("%s <- %s: unexpected resolve response:\nexpected %#v\nbut got  %#v", resolver, request, responses[i], resp.Records[i].String())
		}
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
