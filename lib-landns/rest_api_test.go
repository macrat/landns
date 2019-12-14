package landns_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/macrat/landns/lib-landns"
)

func TestDynamicAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics := landns.NewMetrics("landns")
	resolver, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %s", err)
	}

	s := StartHTTPServer(ctx, t, landns.NewDynamicAPI(resolver).Handler())

	type Test struct {
		Method string
		Path   string
		Body   string
		Status int
		Expect string
	}

	tester := func(tests []Test) func(t *testing.T) {
		return func(t *testing.T) {
			for _, tt := range tests {
				status, got, err := s.Do(tt.Method, tt.Path, tt.Body)
				if err != nil {
					continue
				}
				if status != tt.Status {
					t.Errorf("%s %s: unexpected status code: expected %d but got %d", tt.Method, tt.Path, tt.Status, status)
				}

				if got != tt.Expect {
					t.Errorf("%s %s: unexpected response:\nexpected:\n%s\n\nbut got:\n%s", tt.Method, tt.Path, tt.Expect, got)
				}
			}
		}
	}

	t.Run("success", tester([]Test{
		{"GET", "/v1", "", http.StatusOK, ""},

		{"POST", "/v1", "a.example.com. 42 IN A 127.0.0.1\nb.example.com. 24 IN A 127.0.1.2", http.StatusOK, "; 200: add:2 delete:0\n"},
		{"GET", "/v1", "", http.StatusOK, strings.Join([]string{
			"a.example.com. 42 IN A 127.0.0.1 ; ID:1",
			"1.0.0.127.in-addr.arpa. 42 IN PTR a.example.com. ; ID:2",
			"b.example.com. 24 IN A 127.0.1.2 ; ID:3",
			"2.1.0.127.in-addr.arpa. 24 IN PTR b.example.com. ; ID:4",
			"",
		}, "\n")},
		{"GET", "/v1/suffix/com/example", "", http.StatusOK, "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 24 IN A 127.0.1.2 ; ID:3\n"},
		{"GET", "/v1/suffix/example.com", "", http.StatusOK, "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 24 IN A 127.0.1.2 ; ID:3\n"},

		{"POST", "/v1", "test.com. 100 IN A 127.0.1.1\n;b.example.com. 24 IN A 127.0.1.2", http.StatusOK, "; 200: add:1 delete:1\n"},
		{"GET", "/v1", "", http.StatusOK, strings.Join([]string{
			"a.example.com. 42 IN A 127.0.0.1 ; ID:1",
			"1.0.0.127.in-addr.arpa. 42 IN PTR a.example.com. ; ID:2",
			"test.com. 100 IN A 127.0.1.1 ; ID:5",
			"1.1.0.127.in-addr.arpa. 100 IN PTR test.com. ; ID:6",
			"",
		}, "\n")},
		{"GET", "/v1/suffix/com/example", "", http.StatusOK, "a.example.com. 42 IN A 127.0.0.1 ; ID:1\n"},
		{"GET", "/v1/suffix/example.com", "", http.StatusOK, "a.example.com. 42 IN A 127.0.0.1 ; ID:1\n"},
		{"GET", "/v1/suffix/com", "", http.StatusOK, "a.example.com. 42 IN A 127.0.0.1 ; ID:1\ntest.com. 100 IN A 127.0.1.1 ; ID:5\n"},

		{"DELETE", "/v1", "test.com. 100 IN A 127.0.1.1\n;b.example.com. 24 IN A 127.0.1.2", http.StatusOK, "; 200: add:1 delete:1\n"},
		{"GET", "/v1", "", http.StatusOK, strings.Join([]string{
			"a.example.com. 42 IN A 127.0.0.1 ; ID:1",
			"1.0.0.127.in-addr.arpa. 42 IN PTR a.example.com. ; ID:2",
			"b.example.com. 24 IN A 127.0.1.2 ; ID:7",
			"2.1.0.127.in-addr.arpa. 24 IN PTR b.example.com. ; ID:8",
			"",
		}, "\n")},
		{"GET", "/v1/suffix/com/example", "", http.StatusOK, "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 24 IN A 127.0.1.2 ; ID:7\n"},
		{"GET", "/v1/suffix/example.com", "", http.StatusOK, "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 24 IN A 127.0.1.2 ; ID:7\n"},
	}))

	t.Run("error", tester([]Test{
		{"PATCH", "/v1", "", 405, "; 405: method not allowed\n"},
		{"POST", "/v1/suffix/com", "", 405, "; 405: method not allowed\n"},

		{"GET", "/v1/suffix/com/", "", 404, "; 404: not found\n"},
		{"GET", "/v1/suffix/.com", "", 404, "; 404: not found\n"},
		{"GET", "/v1/suffix/com/.example", "", 404, "; 404: not found\n"},

		{"POST", "/v1", "hello world!\n\ntest", 400, strings.Join([]string{
			"; 400: line 1: invalid format: hello world!",
			";      line 3: invalid format: test",
			"",
		}, "\n")},

		{"DELETE", "/v1", "hello world!\n\ntest", 400, strings.Join([]string{
			"; 400: line 1: invalid format: hello world!",
			";      line 3: invalid format: test",
			"",
		}, "\n")},
	}))
}
