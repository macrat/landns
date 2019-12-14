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
	Resolvers       Resolver
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
	return NewHandler(s.Resolvers, s.Metrics)
}

func (s *Server) ListenAndServe(ctx context.Context, apiAddress *net.TCPAddr, dnsAddress *net.UDPAddr, dnsProto string) error {
	httpHandler, err := s.HTTPHandler()
	if err != nil {
		return err
	}
	httpServer := http.Server{
		Addr:    apiAddress.String(),
		Handler: httpHandler,
	}

	dnsServer := dns.Server{
		Addr:      dnsAddress.String(),
		Net:       dnsProto,
		ReusePort: true,
		Handler:   s.DNSHandler(),
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
		dnsServer.ShutdownContext(ctx)
		return err
	case err = <-dnsch:
		httpServer.Shutdown(ctx)
		return err
	case <-ctx.Done():
		dnsServer.ShutdownContext(ctx)
		httpServer.Shutdown(ctx)
		return nil
	}
}
