package landns

import (
	"log"

	"github.com/miekg/dns"
)

type Handler struct {
	Resolver Resolver
	Metrics  *Metrics
}

func (h Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	end := h.Metrics.Start(r)

	req := Request{RecursionDesired: r.RecursionDesired}
	resp := NewMessageBuilder(r)

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
