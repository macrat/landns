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

type Response struct {
	Records       []Record
	Authoritative bool
}
