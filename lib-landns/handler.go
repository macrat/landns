package landns

import (
	"log"

	"github.com/miekg/dns"
)

// Handler is the implements of dns.Handler of package github.com/miekg/dns.
type Handler struct {
	Resolver           Resolver
	Metrics            *Metrics
	RecursionAvailable bool
}

// NewHandler is constructor of Handler.
func NewHandler(resolver Resolver, metrics *Metrics) Handler {
	return Handler{
		Resolver:           resolver,
		Metrics:            metrics,
		RecursionAvailable: resolver.RecursionAvailable(),
	}
}

// ServeDNS is the method for resolve record.
func (h Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	end := h.Metrics.Start(r)

	req := Request{RecursionDesired: r.RecursionDesired}
	resp := NewMessageBuilder(r, h.RecursionAvailable)

	defer func() {
		msg := resp.Build()
		w.WriteMsg(msg)
		end(msg)
	}()

	if r.Opcode == dns.OpcodeQuery {
		for _, q := range r.Question {
			req.Question = q

			if err := h.Resolver.Resolve(resp, req); err != nil {
				log.Print(err.Error())
				h.Metrics.Error(req, err)
			}
		}
	}
}
