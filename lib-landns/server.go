package landns

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/miekg/dns"
)

// Server is the Landns server instance.
type Server struct {
	Metrics         *Metrics
	DynamicResolver DynamicResolver
	Resolvers       Resolver // Resolvers for this server. Must include DynamicResolver.
}

// HTTPHandler is getter of http.Handler.
func (s *Server) HTTPHandler() (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<h1>Landns</h1><a href="/metrics">metrics</a> <a href="/api/v1">records</a>`)
	})

	metrics, err := s.Metrics.HTTPHandler()
	if err != nil {
		return nil, err
	}

	mux.Handle("/metrics", metrics)
	mux.Handle("/api/", http.StripPrefix("/api", DynamicAPI{s.DynamicResolver}.Handler()))

	return mux, nil
}

// DNSHandler is getter of dns.Handler of package github.com/miekg/dns
func (s *Server) DNSHandler() dns.Handler {
	return NewHandler(s.Resolvers, s.Metrics)
}

// ListenAndServe is starter of server.
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
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpch <- err
		}
	}()
	go func() {
		if err := dnsServer.ListenAndServe(); err != nil {
			dnsch <- err
		}
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
