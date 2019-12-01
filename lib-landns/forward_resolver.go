package landns

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

type ForwardResolver struct {
	client *dns.Client

	Upstreams []*net.UDPAddr
	Metrics   *Metrics
}

func NewForwardResolver(upstreams []*net.UDPAddr, timeout time.Duration, metrics *Metrics) ForwardResolver {
	return ForwardResolver{
		client: &dns.Client{
			Dialer: &net.Dialer{
				Timeout: timeout,
			},
		},
		Upstreams: upstreams,
		Metrics:   metrics,
	}
}

func (fr ForwardResolver) Resolve(w ResponseWriter, r Request) error {
	if !r.RecursionDesired {
		return nil
	}

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{
			{Name: r.Name, Qtype: r.Qtype, Qclass: r.Qclass},
		},
	}

	for _, upstream := range fr.Upstreams {
		in, rtt, err := fr.client.Exchange(msg, upstream.String())
		if err != nil {
			continue
		}
		fr.Metrics.UpstreamTime(rtt)
		for _, answer := range in.Answer {
			record, err := NewRecordFromRR(answer)
			if err != nil {
				return err
			}
			w.SetNoAuthoritative()
			w.Add(record)
		}
		break
	}

	return nil
}

func (fr ForwardResolver) RecursionAvailable() bool {
	return true
}
