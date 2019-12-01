package landns

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/miekg/dns"
)

type Server struct {
	Metrics         *Metrics
	DynamicResolver DynamicResolver
	StaticResolver  Resolver
}

func (s *Server) HTTPHandler() (http.Handler, error) {
	mux := http.NewServeMux()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<h1>Landns</h1><a href=\"/metrics\">metrics</a>")
	})

	metrics, err := s.Metrics.HTTPHandler()
	if err != nil {
		return nil, err
	}

	mux.Handle("/metrics", metrics)
	mux.Handle("/api/", http.StripPrefix("/api", NewDynamicAPI(s.DynamicResolver).Handler()))

	return mux, nil
}

func (s *Server) DNSHandler() dns.Handler {
	return Handler{
		Resolver: ResolverSet{s.DynamicResolver, s.StaticResolver},
		Metrics:  s.Metrics,
	}
}

func (s *Server) ListenAndServe(apiAddress, dnsAddress *net.TCPAddr, dnsProto string) error {
	httpHandler, err := s.HTTPHandler()
	if err != nil {
		return err
	}
	httpServer := http.Server{
		Addr:    apiAddress.String(),
		Handler: httpHandler,
	}

	dnsServer := dns.Server{
		Addr:    dnsAddress.String(),
		Net:     dnsProto,
		Handler: s.DNSHandler(),
	}

	httpch := make(chan error)
	dnsch := make(chan error)
	defer close(httpch)
	defer close(dnsch)

	go func() {
		httpch <- httpServer.ListenAndServe()
	}()
	go func() {
		dnsch <- dnsServer.ListenAndServe()
	}()

	select {
	case err = <-httpch:
		dnsServer.Shutdown()
		return err
	case err = <-dnsch:
		httpServer.Shutdown(context.Background())
		return err
	}
}
