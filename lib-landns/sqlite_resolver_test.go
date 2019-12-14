package landns_test

import (
	"fmt"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func CreateSqliteResolver(t testing.TB) *landns.SqliteResolver {
	t.Helper()

	metrics := landns.NewMetrics("landns")
	resolver, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		t.Fatalf("failed to make sqlite resolver: %s", err.Error())
	}

	return resolver
}

func TestSqliteResolver(t *testing.T) {
	resolver := CreateSqliteResolver(t)
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatalf("failed to close: %s", err)
		}
	}()

	if s := resolver.String(); s != "SqliteResolver[:memory:]" {
		t.Errorf(`unexpected string: expected "SqliteResolver[:memory:]" but got %#v`, s)
	}

	if resolver.RecursionAvailable() != false {
		t.Errorf("unexpected recursion available: %#v", resolver.RecursionAvailable())
	}

	type Test struct {
		Request       landns.Request
		Authoritative bool
		Expect        []string
	}
	tests := []struct {
		Records string
		Tests   []Test
		Entries map[int]string
		Suffix  map[landns.Domain][]string
		Glob    map[string][]string
	}{
		{
			Records: `
				example.com. 42 IN A 127.0.0.1
				example.com. 100 IN A 127.0.0.2
				example.com. 200 IN AAAA 4::2
				example.com. 300 IN TXT "hello world"
				abc.example.com. 400 IN CNAME example.com.
				abc.example.com. 400 IN CNAME example.com.
			`,
			Tests: []Test{
				{landns.NewRequest("example.com.", dns.TypeA, false), true, []string{"example.com. 42 IN A 127.0.0.1", "example.com. 100 IN A 127.0.0.2"}},
				{landns.NewRequest("example.com.", dns.TypeAAAA, false), true, []string{"example.com. 200 IN AAAA 4::2"}},
				{landns.NewRequest("example.com.", dns.TypeTXT, false), true, []string{"example.com. 300 IN TXT \"hello world\""}},
				{landns.NewRequest("abc.example.com.", dns.TypeCNAME, false), true, []string{"abc.example.com. 400 IN CNAME example.com."}},
				{landns.NewRequest("1.0.0.127.in-addr.arpa.", dns.TypePTR, false), true, []string{"1.0.0.127.in-addr.arpa. 42 IN PTR example.com."}},
			},
			Entries: map[int]string{
				1: "example.com. 42 IN A 127.0.0.1 ; ID:1",
				2: "1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
				3: "example.com. 100 IN A 127.0.0.2 ; ID:3",
				4: "2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				5: "example.com. 200 IN AAAA 4::2 ; ID:5",
				6: "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				7: "example.com. 300 IN TXT \"hello world\" ; ID:7",
				8: "abc.example.com. 400 IN CNAME example.com. ; ID:8",
			},
			Suffix: map[landns.Domain][]string{
				"example.com.": {
					"example.com. 42 IN A 127.0.0.1 ; ID:1",
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 200 IN AAAA 4::2 ; ID:5",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				},
				"abc.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				},
				"in-addr.arpa.": {
					"1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				},
				"arpa.": {
					"1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				},
			},
			Glob: map[string][]string{
				"*.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				},
				"2*arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				},
			},
		},
		{
			Records: `
				;example.com. 42 IN A 127.0.0.1
				new.example.com. 42 IN A 127.0.1.1
			`,
			Tests: []Test{
				{landns.NewRequest("example.com.", dns.TypeA, false), true, []string{"example.com. 100 IN A 127.0.0.2"}},
				{landns.NewRequest("example.com.", dns.TypeAAAA, false), true, []string{"example.com. 200 IN AAAA 4::2"}},
				{landns.NewRequest("example.com.", dns.TypeTXT, false), true, []string{"example.com. 300 IN TXT \"hello world\""}},
				{landns.NewRequest("abc.example.com.", dns.TypeCNAME, false), true, []string{"abc.example.com. 400 IN CNAME example.com."}},
				{landns.NewRequest("1.0.0.127.in-addr.arpa.", dns.TypePTR, false), true, []string{}},
				{landns.NewRequest("new.example.com.", dns.TypeA, false), true, []string{"new.example.com. 42 IN A 127.0.1.1"}},
				{landns.NewRequest("1.1.0.127.in-addr.arpa.", dns.TypePTR, false), true, []string{"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com."}},
			},
			Entries: map[int]string{
				3:  "example.com. 100 IN A 127.0.0.2 ; ID:3",
				4:  "2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				5:  "example.com. 200 IN AAAA 4::2 ; ID:5",
				6:  "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				7:  "example.com. 300 IN TXT \"hello world\" ; ID:7",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "new.example.com. 42 IN A 127.0.1.1 ; ID:9",
				10: "1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:10",
			},
			Suffix: map[landns.Domain][]string{
				"example.com.": {
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 200 IN AAAA 4::2 ; ID:5",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:9",
				},
				"abc.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				},
				"in-addr.arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:10",
				},
				"arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:10",
				},
			},
			Glob: map[string][]string{
				"*example.com.": {
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 200 IN AAAA 4::2 ; ID:5",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:9",
				},
				"*.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:9",
				},
				"2*arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				},
			},
		},
		{
			Records: `
				;example.com. 42 IN A 127.0.0.1 ; ID:1
				;example.com. 200 IN AAAA 4::2 ; ID:5
				;no.example.com. 200 IN AAAA 4::2 ; ID:5
			`,
			Tests: []Test{
				{landns.NewRequest("example.com.", dns.TypeA, false), true, []string{"example.com. 100 IN A 127.0.0.2"}},
				{landns.NewRequest("example.com.", dns.TypeAAAA, false), true, []string{}},
			},
			Entries: map[int]string{
				3:  "example.com. 100 IN A 127.0.0.2 ; ID:3",
				4:  "2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				7:  "example.com. 300 IN TXT \"hello world\" ; ID:7",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "new.example.com. 42 IN A 127.0.1.1 ; ID:9",
				10: "1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:10",
			},
			Suffix: map[landns.Domain][]string{
				"example.com.": {
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:9",
				},
				"abc.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				},
				"in-addr.arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:10",
				},
				"arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:10",
				},
			},
			Glob: map[string][]string{
				"*.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:9",
				},
				"2*arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				},
			},
		},
	}

	for _, test := range tests {
		records, err := landns.NewDynamicRecordSet(test.Records)
		if err != nil {
			t.Fatalf("failed to make dynamic records: %s", err)
		}

		if err := resolver.SetRecords(records); err != nil {
			t.Errorf("failed to set records: %s", err)
		}

		for id, expect := range test.Entries {
			record, err := resolver.GetRecord(id)
			if err != nil {
				t.Errorf("failed to get record: %d: %d", id, err)
				continue
			}

			if record.String() != expect+"\n" {
				t.Errorf("failed to get record: %d:\nexpected: %#v\nbut got:  %#v", id, expect+"\n", record.String())
			}
		}

		ttfuncs := []func() ([]string, landns.DynamicRecordSet, error){
			func() ([]string, landns.DynamicRecordSet, error) {
				rs, err := resolver.Records()
				es := make([]string, len(test.Entries))
				ptr := 0
				for i := 0; ptr < len(test.Entries); i++ {
					x, ok := test.Entries[i]
					if ok {
						es[ptr] = x
						ptr++
					}
				}
				return es, rs, err
			},
		}
		for suffix, expect := range test.Suffix {
			ttfuncs = append(ttfuncs, func() ([]string, landns.DynamicRecordSet, error) {
				rs, err := resolver.SearchRecords(suffix)
				return expect, rs, err
			})
		}
		for glob, expect := range test.Glob {
			ttfuncs = append(ttfuncs, func() ([]string, landns.DynamicRecordSet, error) {
				rs, err := resolver.GlobRecords(glob)
				return expect, rs, err
			})
		}

		for _, ttfunc := range ttfuncs {
			expect, got, err := ttfunc()
			if err != nil {
				t.Errorf("failed to get records: %s", err)
				continue
			}

			ok := len(expect) == len(got)
			if ok {
				for i := range got {
					if got[i].String() != expect[i] {
						ok = false
					}
				}
			}
			if !ok {
				txt := "unexpected entries:\nexpected:\n"
				for _, t := range expect {
					txt += "\t" + t + "\n"
				}
				txt += "\nbut got:\n"

				for _, r := range got {
					txt += "\t" + r.String() + "\n"
				}
				t.Errorf(txt)
			}
		}

		for _, tt := range test.Tests {
			AssertResolve(t, resolver, tt.Request, tt.Authoritative, tt.Expect...)
		}
	}
}

func BenchmarkSqliteResolver(b *testing.B) {
	resolver := CreateSqliteResolver(b)
	defer func() {
		if err := resolver.Close(); err != nil {
			b.Fatalf("failed to close: %s", err)
		}
	}()

	records := make(landns.DynamicRecordSet, 200)

	var err error
	for i := 0; i < 100; i++ {
		records[i*2], err = landns.NewDynamicRecord(fmt.Sprintf("host%d.example.com. 0 IN A 127.1.2.3", i))
		if err != nil {
			b.Fatalf("failed to make dynamic record: %v", err)
		}

		records[i*2+1], err = landns.NewDynamicRecord(fmt.Sprintf("host%d.example.com. 0 IN A 127.2.3.4", i))
		if err != nil {
			b.Fatalf("failed to make dynamic record: %v", err)
		}
	}

	resolver.SetRecords(records)

	req := landns.NewRequest("host50.example.com.", dns.TypeA, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resolver.Resolve(NewDummyResponseWriter(), req)
	}

	b.StopTimer()
}
