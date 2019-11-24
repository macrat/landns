package landns

import (
	"log"

	"github.com/miekg/dns"
)

type Handler struct {
	Resolver Resolver
	Metrics  *Metrics
}

func (h Handler) answers(q dns.Question) (answers []dns.RR) {
	responses, err := h.Resolver.Resolve(q)
	if err != nil {
		log.Print(err.Error())
		h.Metrics.Error(q, err)
		return
	}

	for _, resp := range responses {
		if rr, err := resp.ToRR(); err != nil {
			log.Print(err.Error())
			h.Metrics.Error(q, err)
		} else {
			answers = append(answers, rr)
		}
	}

	return
}

func (h Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	end := h.Metrics.Start(r)

	m := new(dns.Msg)
	m.SetReply(r)

	defer end(m)

	if r.Opcode != dns.OpcodeQuery {
		w.WriteMsg(m)
		return
	}

	for _, q := range m.Question {
		m.Answer = append(m.Answer, dns.Dedup(h.answers(q), nil)...)
	}

	w.WriteMsg(m)
}
