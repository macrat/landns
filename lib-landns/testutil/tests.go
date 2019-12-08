package testutil

import (
	"sort"
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns"
)

func AssertResolve(t *testing.T, resolver landns.Resolver, request landns.Request, authoritative bool, responses ...string) {
	resp := NewDummyResponseWriter()
	if err := resolver.Resolve(resp, request); err != nil {
		t.Errorf("%s <- %s: failed to resolve: %v", resolver, request, err.Error())
		return
	}

	if resp.Authoritative != authoritative {
		t.Errorf(`%s <- %s: unexpected authoritive of response: expected %v but got %v`, resolver, request, authoritative, resp.Authoritative)
	}

	if len(resp.Records) != len(responses) {
		t.Errorf(`%s <- %s: unexpected resolve response: expected length %d but got %d`, resolver, request, len(responses), len(resp.Records))
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
			t.Errorf(`%s <- %s: unexpected resolve response: expected "%s" but got "%s"`, resolver, request, responses[i], resp.Records[i])
		}
	}
}
