package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/macrat/landns/lib-landns"
	"github.com/miekg/dns"
)

func MakeDummyFile(content string) (closer func(), path string, err error) {
	tmp, err := ioutil.TempFile("", "landns_test_")
	if err != nil {
		return nil, "", err
	}

	fmt.Fprint(tmp, content)

	return func() {
		os.Remove(tmp.Name())
	}, tmp.Name(), nil
}

func TestLoadStaticResolvers(t *testing.T) {
	closer, pathA, err := MakeDummyFile(`ttl: 10
address:
  example.com.: [127.0.1.2]`)
	if err != nil {
		t.Fatalf("failed to make dummy file: %s", err)
	}
	defer closer()

	closer, pathB, err := MakeDummyFile(`ttl: 20
address:
  example.com.: [127.1.2.3]`)
	if err != nil {
		t.Fatalf("failed to make dummy file: %s", err)
	}
	defer closer()

	resolver, err := loadStatisResolvers([]string{pathA, pathB})
	if err != nil {
		t.Fatalf("failed to load configs: %s", err)
	}

	if len(resolver) != 2 {
		t.Fatalf("unexpected length of resolver set: expected 2 but got %d", len(resolver))
	}

	records := []landns.Record{}
	writer := landns.NewResponseCallback(func(r landns.Record) error {
		records = append(records, r)
		return nil
	})
	err = resolver.Resolve(writer, landns.NewRequest("example.com.", dns.TypeA, false))
	if err != nil {
		t.Fatalf("failed to resolve: %s", err)
	}

	if len(records) != 2 {
		t.Fatalf("unexpected response length: expected 2 but got %d", len(records))
	}

	if records[0].String() != "example.com. 10 IN A 127.0.1.2" {
		t.Errorf("unexpected response: %s", records[0])
	}

	if records[1].String() != "example.com. 20 IN A 127.1.2.3" {
		t.Errorf("unexpected response: %s", records[1])
	}
}
