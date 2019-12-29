package landns_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/macrat/landns/lib-landns"
	"github.com/macrat/landns/lib-landns/testutil"
	"github.com/miekg/dns"
)

func TestDynamicRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input  string
		Expect string
		Error  string
	}{
		{"example.com. 42 IN A 127.0.1.2 ; ID:123", "example.com. 42 IN A 127.0.1.2 ; ID:123", ""},
		{"example.com. 123 IN A 127.3.4.5 ; dummy:aa ID:67 foo:bar", "example.com. 123 IN A 127.3.4.5 ; ID:67", ""},
		{"hello 100 IN A 127.6.7.8", "hello. 100 IN A 127.6.7.8", ""},
		{"v6.example.com. 321 IN AAAA 4::2", "v6.example.com. 321 IN AAAA 4::2", ""},
		{"example.com.\t135\tIN\tTXT\thello\t;\tID:1", "example.com. 135 IN TXT \"hello\" ; ID:1", ""},
		{"c.example.com. IN CNAME example.com. ; ID:2", "c.example.com. 3600 IN CNAME example.com. ; ID:2", ""},
		{"_web._tcp.example.com. SRV 1 2 3 example.com. ; ID:4", "_web._tcp.example.com. 3600 IN SRV 1 2 3 example.com. ; ID:4", ""},
		{"2.1.0.127.in-arpa.addr. 2 IN PTR example.com. ; ID:987654321", "2.1.0.127.in-arpa.addr. 2 IN PTR example.com. ; ID:987654321", ""},
		{"; disabled.com. 100 IN A 127.1.2.3", ";disabled.com. 100 IN A 127.1.2.3", ""},
		{";disabled.com. 100 IN A 127.1.2.3 ; ID:4", ";disabled.com. 100 IN A 127.1.2.3 ; ID:4", ""},
		{"volatile.com. 100 IN A 127.1.2.3 ; Volatile", "volatile.com. 100 IN A 127.1.2.3 ; Volatile", ""},
		{";disabled.volatile.com. 100 IN A 127.1.2.3 ; Id:5 VOLATILE", ";disabled.volatile.com. 100 IN A 127.1.2.3 ; ID:5 Volatile", ""},
		{"a\nb", "", landns.ErrMultiLineDynamicRecord.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID: 42", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID:foobar", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"hello world ; ID:1", "", `failed to parse record: dns: not a TTL: "world" at line: 1:12`},
	}

	for _, tt := range tests {
		r, err := landns.NewDynamicRecord(tt.Input)
		if err != nil && tt.Error == "" {
			t.Errorf("failed to unmarshal dynamic record: %v", err)
			continue
		} else if err != nil && err.Error() != tt.Error {
			t.Errorf(`unmarshal dynamic record: expected error "%v" but got "%v"`, tt.Error, err)
			continue
		}
		if tt.Error != "" {
			continue
		}

		if got, err := r.MarshalText(); err != nil {
			t.Errorf("failed to marshal dynamic record: %v", err)
		} else if string(got) != tt.Expect {
			t.Errorf("encoded text was unexpected:\n\texpected: %#v\n\tbut got:  %#v", tt.Expect, string(got))
		}
	}
}

func TestDynamicRecordSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input  string
		Expect string
		Error  string
	}{
		{"example.com. 42 IN A 127.0.1.2 ; ID:3", "example.com. 42 IN A 127.0.1.2 ; ID:3\n", ""},
		{"example.com. 42 IN A 127.0.1.2 ; ID:3\nexample.com. 24 IN AAAA 1:2:3::4", "example.com. 42 IN A 127.0.1.2 ; ID:3\nexample.com. 24 IN AAAA 1:2:3::4\n", ""},
		{"\n\n\nexample.com. 42 IN A 127.0.1.2 ; ID:3\n\n", "example.com. 42 IN A 127.0.1.2 ; ID:3\n", ""},
		{";this\n  ;is\n\t; comment", "", ""},
		{"unexpected\nexample.com. 1 IN A 127.1.2.3\n\naa", "", "line 1: invalid format: unexpected\nline 4: invalid format: aa"},
	}

	for _, tt := range tests {
		rs, err := landns.NewDynamicRecordSet(tt.Input)
		if err != nil && tt.Error == "" {
			t.Errorf("failed to unmarshal dynamic record set: %v", err)
			continue
		} else if err != nil && err.Error() != tt.Error {
			t.Errorf(`unmarshal dynamic record set: expected error "%v" but got "%v"`, tt.Error, err)
			continue
		}
		if tt.Error != "" {
			continue
		}

		if got, err := rs.MarshalText(); err != nil {
			t.Errorf("failed to marshal dynamic record set: %v", err)
		} else if string(got) != tt.Expect {
			t.Errorf("encoded text was unexpected:\n\texpected: %#v\n\tbut got:  %#v", tt.Expect, string(got))
		}
	}
}

func DynamicResolverTest(t *testing.T, resolver landns.DynamicResolver) {
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
		Delete  []int
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
				example.com. 500 IN MX 10 mx.example.com.
				example.com. IN NS ns1.example.com.
			`,
			Tests: []Test{
				{landns.NewRequest("example.com.", dns.TypeA, false), true, []string{"example.com. 42 IN A 127.0.0.1", "example.com. 100 IN A 127.0.0.2"}},
				{landns.NewRequest("example.com.", dns.TypeAAAA, false), true, []string{"example.com. 200 IN AAAA 4::2"}},
				{landns.NewRequest("example.com.", dns.TypeTXT, false), true, []string{"example.com. 300 IN TXT \"hello world\""}},
				{landns.NewRequest("abc.example.com.", dns.TypeCNAME, false), true, []string{"abc.example.com. 400 IN CNAME example.com."}},
				{landns.NewRequest("1.0.0.127.in-addr.arpa.", dns.TypePTR, false), true, []string{"1.0.0.127.in-addr.arpa. 42 IN PTR example.com."}},
				{landns.NewRequest("example.com.", dns.TypeMX, false), true, []string{"example.com. 500 IN MX 10 mx.example.com."}},
				{landns.NewRequest("example.com.", dns.TypeNS, false), true, []string{"example.com. IN NS ns1.example.com."}},
			},
			Entries: map[int]string{
				1:  "example.com. 42 IN A 127.0.0.1 ; ID:1",
				2:  "1.0.0.127.in-addr.arpa. 42 IN PTR example.com. ; ID:2",
				3:  "example.com. 100 IN A 127.0.0.2 ; ID:3",
				4:  "2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				5:  "example.com. 200 IN AAAA 4::2 ; ID:5",
				6:  "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
				7:  "example.com. 300 IN TXT \"hello world\" ; ID:7",
				8:  "abc.example.com. 400 IN CNAME example.com. ; ID:8",
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
			},
			Suffix: map[landns.Domain][]string{
				"example.com.": {
					"example.com. 42 IN A 127.0.0.1 ; ID:1",
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 200 IN AAAA 4::2 ; ID:5",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
					"example.com. IN NS ns1.example.com. ; ID:10",
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
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
				11: "new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				12: "1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
			},
			Suffix: map[landns.Domain][]string{
				"example.com.": {
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 200 IN AAAA 4::2 ; ID:5",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
					"example.com. IN NS ns1.example.com. ; ID:10",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				},
				"abc.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				},
				"in-addr.arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
				},
				"arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.4.0.0.0.ip6.arpa. 200 IN PTR example.com. ; ID:6",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
				},
			},
			Glob: map[string][]string{
				"*example.com.": {
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 200 IN AAAA 4::2 ; ID:5",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
					"example.com. IN NS ns1.example.com. ; ID:10",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				},
				"*.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:11",
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
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				10: "example.com. IN NS ns1.example.com. ; ID:10",
				11: "new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				12: "1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
			},
			Suffix: map[landns.Domain][]string{
				"example.com.": {
					"example.com. 100 IN A 127.0.0.2 ; ID:3",
					"example.com. 300 IN TXT \"hello world\" ; ID:7",
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"example.com. 500 IN MX 10 mx.example.com. ; ID:9",
					"example.com. IN NS ns1.example.com. ; ID:10",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				},
				"abc.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
				},
				"in-addr.arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
				},
				"arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
					"1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
				},
			},
			Glob: map[string][]string{
				"*.example.com.": {
					"abc.example.com. 400 IN CNAME example.com. ; ID:8",
					"new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				},
				"2*arpa.": {
					"2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				},
			},
		},
		{
			Delete: []int{3, 8, 10},
			Entries: map[int]string{
				4:  "2.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:4",
				7:  "example.com. 300 IN TXT \"hello world\" ; ID:7",
				9:  "example.com. 500 IN MX 10 mx.example.com. ; ID:9",
				11: "new.example.com. 42 IN A 127.0.1.1 ; ID:11",
				12: "1.1.0.127.in-addr.arpa. 42 IN PTR new.example.com. ; ID:12",
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

		for _, id := range test.Delete {
			if err := resolver.RemoveRecord(id); err != nil {
				t.Errorf("failed to delete record: %s", err)
			}
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

		ttfuncs := []func() (string, []string, landns.DynamicRecordSet, error){
			func() (string, []string, landns.DynamicRecordSet, error) {
				rs, err := resolver.Records()
				sort.SliceStable(rs, func(i, j int) bool {
					return *rs[i].ID < *rs[j].ID
				})
				es := make([]string, len(test.Entries))
				ptr := 0
				for i := 0; ptr < len(test.Entries); i++ {
					x, ok := test.Entries[i]
					if ok {
						es[ptr] = x
						ptr++
					}
				}
				return "Records", es, rs, err
			},
		}
		for suffix, expect := range test.Suffix {
			ttfuncs = append(ttfuncs, func() (string, []string, landns.DynamicRecordSet, error) {
				rs, err := resolver.SearchRecords(suffix)
				sort.SliceStable(rs, func(i, j int) bool {
					return *rs[i].ID < *rs[j].ID
				})
				return "SearchRecords", expect, rs, err
			})
		}
		for glob, expect := range test.Glob {
			ttfuncs = append(ttfuncs, func() (string, []string, landns.DynamicRecordSet, error) {
				rs, err := resolver.GlobRecords(glob)
				sort.SliceStable(rs, func(i, j int) bool {
					return *rs[i].ID < *rs[j].ID
				})
				return "GlobRecords", expect, rs, err
			})
		}

		for _, ttfunc := range ttfuncs {
			name, expect, got, err := ttfunc()
			if err != nil {
				t.Errorf("failed to get records: %s", err)
				continue
			}

			AssertDynamicRecordSet(t, name, expect, got)
		}

		for _, tt := range test.Tests {
			AssertResolve(t, resolver, tt.Request, tt.Authoritative, tt.Expect...)
		}
	}
}

func DynamicResolverTest_Volatile(t *testing.T, resolver landns.DynamicResolver) {
	records, err := landns.NewDynamicRecordSet(`
		fixed.example.com. 100 IN TXT "fixed"
		long.example.com. 100 IN TXT "long" ; Volatile
		short.example.com. 1 IN TXT "short" ; Volatile
	`)
	if err != nil {
		t.Fatalf("failed to make dynamic records: %s", err)
	}

	if err := resolver.SetRecords(records); err != nil {
		t.Errorf("failed to set records: %s", err)
	}

	time.Sleep(1500 * time.Millisecond)

	rs, err := resolver.Records()
	if err != nil {
		t.Errorf("failed to get records: %s", err)
	}
	AssertDynamicRecordSet(t, "volatile records", []string{
		`fixed.example.com. 100 IN TXT "fixed" ; ID:1`,
		`long.example.com. 98 IN TXT "long" ; ID:2 Volatile`,
	}, rs)

	AssertResolve(t, resolver, landns.NewRequest("long.example.com.", dns.TypeTXT, false), true, `long.example.com. 98 IN TXT "long"`)
}

func DynamicResolverBenchmark(b *testing.B, resolver landns.DynamicResolver) {
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
		resolver.Resolve(testutil.NewDummyResponseWriter(), req)
	}

	b.StopTimer()
}

func ExampleDynamicRecord() {
	record, _ := landns.NewDynamicRecord("example.com. 600 IN A 127.0.0.1")
	fmt.Println("name:", record.Record.GetName(), "disabled:", record.Disabled)

	record, _ = landns.NewDynamicRecord(";test.service 300 IN TXT \"hello world\"")
	fmt.Println("name:", record.Record.GetName(), "disabled:", record.Disabled)

	// Output:
	// name: example.com. disabled: false
	// name: test.service. disabled: true
}

func ExampleDynamicRecord_String() {
	record, _ := landns.NewDynamicRecord("example.com. 600 IN A 127.0.0.1")

	fmt.Println(record)

	record.Disabled = true
	fmt.Println(record)

	id := 10
	record.ID = &id
	fmt.Println(record)

	// Output:
	// example.com. 600 IN A 127.0.0.1
	// ;example.com. 600 IN A 127.0.0.1
	// ;example.com. 600 IN A 127.0.0.1 ; ID:10
}

func ExampleDynamicRecordSet() {
	records, _ := landns.NewDynamicRecordSet(`
	a.example.com. 100 IN A 127.0.0.1
	b.example.com. 200 IN A 127.0.1.2
`)

	for _, r := range records {
		fmt.Println(r.Record.GetName())
		fmt.Println(r)
	}

	// Output:
	// a.example.com.
	// a.example.com. 100 IN A 127.0.0.1
	// b.example.com.
	// b.example.com. 200 IN A 127.0.1.2
}
