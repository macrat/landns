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
	return Request{dns.Question{Name: name, Qtype: qtype, Qclass: dns.ClassINET}, recursionDesired}
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

type ResponseCallback struct {
	Callback      func(Record) error
	Authoritative bool
}

func NewResponseCallback(callback func(Record) error) *ResponseCallback {
	return &ResponseCallback{Callback: callback, Authoritative: true}
}

func (rc *ResponseCallback) Add(r Record) error {
	return rc.Callback(r)
}

func (rc *ResponseCallback) IsAuthoritative() bool {
	return rc.Authoritative
}

func (rc *ResponseCallback) SetNoAuthoritative() {
	rc.Authoritative = false
}

type MessageBuilder struct {
	request            *dns.Msg
	records            []dns.RR
	authoritative      bool
	recursionAvailable bool
}

func NewMessageBuilder(request *dns.Msg, recursionAvailable bool) *MessageBuilder {
	return &MessageBuilder{
		request:            request,
		records:            make([]dns.RR, 0, 10),
		authoritative:      true,
		recursionAvailable: recursionAvailable,
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

	msg.Authoritative = mb.authoritative
	msg.RecursionAvailable = mb.recursionAvailable

	return msg
}
