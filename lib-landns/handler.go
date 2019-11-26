package landns

import (
	"log"

	"github.com/miekg/dns"
)

type Handler struct {
	Resolver Resolver
	Metrics  *Metrics
}

func (h Handler) answers(req Request) (answers []dns.RR, authoritative bool) {
	response, err := h.Resolver.Resolve(req)
	if err != nil {
		log.Print(err.Error())
		h.Metrics.Error(req, err)
		return
	}

	authoritative = response.Authoritative

	for _, resp := range response.Records {
		if rr, err := resp.ToRR(); err != nil {
			log.Print(err.Error())
			h.Metrics.Error(req, err)
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

	req := Request{RecursionDesired: r.RecursionDesired}
	m.Authoritative = true

	for _, q := range m.Question {
		req.Question = q

		answers, authoritative := h.answers(req)
		m.Answer = append(m.Answer, dns.Dedup(answers, nil)...)
		if !authoritative {
			m.Authoritative = false
		}
	}

	w.WriteMsg(m)
}
