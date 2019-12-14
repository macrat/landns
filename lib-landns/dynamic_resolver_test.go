package landns_test

import (
	"testing"

	"github.com/macrat/landns/lib-landns"
)

func TestDynamicRecord(t *testing.T) {
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
		{"a\nb", "", landns.ErrMultiLineDynamicRecord.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID: 42", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"example.com. 42 IN A 127.0.1.2 ; ID:foobar", "", landns.ErrInvalidDynamicRecordFormat.Error()},
		{"hello world ; ID:1", "", `dns: not a TTL: "world" at line: 1:12`},
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
