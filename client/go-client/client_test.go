package client_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/macrat/landns/client/go-client"
	"github.com/macrat/landns/lib-landns"
)

func StartServer(ctx context.Context) (*url.URL, error) {
	metrics := landns.NewMetrics("landns")
	resolver, err := landns.NewSqliteResolver(":memory:", metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to make sqlite resolver: %s", err)
	}

	server := landns.Server{
		Metrics:         metrics,
		DynamicResolver: resolver,
		Resolvers:       resolver,
	}
	go func() {
		err := server.ListenAndServe(ctx, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9353}, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3553}, "udp")
		if err != nil {
			panic(fmt.Sprintf("failed to start server: %s", err))
		}
	}()
	time.Sleep(10 * time.Millisecond)
	u, err := url.Parse("http://127.0.0.1:9353/api/v1/")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %s", err)
	}

	return u, nil
}

func Example() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	apiAddress, _ := StartServer(ctx) // Start Landns server for test. (this is debug function)

	fmt.Println("api address:", apiAddress)
	fmt.Println()

	c := client.New(apiAddress)

	rs, err := landns.NewDynamicRecordSet(`
	example.com. 100 IN A 127.0.0.1
	example.com. 200 IN A 127.0.0.2
`)
	if err != nil {
		panic(err.Error())
	}

	err = c.Set(rs) // Register records.
	if err != nil {
		panic(err.Error())
	}

	rs, err = c.Get() // Get all records.
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("all:")
	fmt.Print(rs)
	fmt.Println()

	rs, err = c.Glob("*.com") // Get records that ends with ".com".
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("glob:")
	fmt.Print(rs)
	fmt.Println()

	// Output:
	// api address: http://127.0.0.1:9353/api/v1/
	//
	// all:
	// example.com. 100 IN A 127.0.0.1 ; ID:1
	// 1.0.0.127.in-addr.arpa. 100 IN PTR example.com. ; ID:2
	// example.com. 200 IN A 127.0.0.2 ; ID:3
	// 2.0.0.127.in-addr.arpa. 200 IN PTR example.com. ; ID:4
	//
	// glob:
	// example.com. 100 IN A 127.0.0.1 ; ID:1
	// example.com. 200 IN A 127.0.0.2 ; ID:3
}

func TestAPIClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	u, err := StartServer(ctx)
	if err != nil {
		t.Fatalf(err.Error())
	}
	client := client.New(u)

	rs, err := landns.NewDynamicRecordSet(`a.example.com. 42 IN A 127.0.0.1 ; ID:1
1.0.0.127.in-addr.arpa. 42 IN PTR a.example.com. ; ID:2
b.example.com. 100 IN A 127.1.2.3 ; ID:3
3.2.1.127.in-addr.arpa. 100 IN PTR b.example.com. ; ID:4`)
	if err != nil {
		t.Fatalf("failed to parse records: %s", err)
	}

	if err := client.Set(rs); err != nil {
		t.Fatalf("failed to set records: %s", err)
	}

	if resp, err := client.Get(); err != nil {
		t.Fatalf("failed to get records: %s", err)
	} else if resp.String() != rs.String() {
		t.Fatalf("unexpected get response:\nexpect:\n%s\nbut got:\n%s", rs, resp)
	}

	expect := "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 100 IN A 127.1.2.3 ; ID:3\n"
	if resp, err := client.Glob("*.example.com"); err != nil {
		t.Fatalf("failed to glob records: %s", err)
	} else if resp.String() != expect {
		t.Fatalf("unexpected glob response:\nexpect:\n%s\nbut got:\n%s", expect, resp)
	}

	if err := client.Remove(2); err != nil {
		t.Fatalf("failed to remove records: %s", err)
	}

	expect = "a.example.com. 42 IN A 127.0.0.1 ; ID:1\nb.example.com. 100 IN A 127.1.2.3 ; ID:3\n3.2.1.127.in-addr.arpa. 100 IN PTR b.example.com. ; ID:4\n"
	if resp, err := client.Get(); err != nil {
		t.Fatalf("failed to glob records: %s", err)
	} else if resp.String() != expect {
		t.Fatalf("unexpected glob response:\nexpect:\n%s\nbut got:\n%s", expect, resp)
	}
}
