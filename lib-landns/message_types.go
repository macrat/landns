package landns

import (
	"fmt"

	"github.com/miekg/dns"
)

type Request struct {
	dns.Question

	RecursionDesired bool
}

func NewRequest(name string, qtype uint16, recursionDesired bool) Request {
	return Request{dns.Question{Name: name, Qtype: qtype}, recursionDesired}
}

func (req Request) QtypeString() string {
	switch req.Qtype {
	case dns.TypeA:
		return "A"
	case dns.TypeNS:
		return "NS"
	case dns.TypeCNAME:
		return "CNAME"
	case dns.TypePTR:
		return "PTR"
	case dns.TypeMX:
		return "MX"
	case dns.TypeTXT:
		return "TXT"
	case dns.TypeAAAA:
		return "AAAA"
	case dns.TypeSRV:
		return "SRV"
	default:
		return ""
	}
}

func (req Request) String() string {
	return fmt.Sprintf("%s %s", req.Name, req.QtypeString())
}

type ResponseWriter interface {
	Add(Record) error
	IsAuthoritative() bool
	SetNoAuthoritative()
}

type MessageBuilder struct {
	request       *dns.Msg
	records       []dns.RR
	authoritative bool
}

func NewMessageBuilder(request *dns.Msg) *MessageBuilder {
	return &MessageBuilder{
		request: request,
		records: make([]dns.RR, 0, 10),
	}
}

func (mb *MessageBuilder) Add(r Record) error {
	rr, err := r.ToRR()
	if err != nil {
		return err
	}

	mb.records = append(mb.records, rr)
	return nil
}

func (mb *MessageBuilder) IsAuthoritative() bool {
	return mb.authoritative
}

func (mb *MessageBuilder) SetNoAuthoritative() {
	mb.authoritative = false
}

func (mb *MessageBuilder) Build() *dns.Msg {
	msg := new(dns.Msg)
	msg.SetReply(mb.request)

	msg.Answer = dns.Dedup(mb.records, nil)

	return msg
}
