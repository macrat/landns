package landns

import (
	"strconv"
	"bytes"
	"fmt"

	"github.com/miekg/dns"
)

var (
	ErrMultiLineDynamicRecord = fmt.Errorf("DynamicRecord can't have multi line")
	ErrInvalidDynamicRecordFormat = fmt.Errorf("DynamicRecord invalid format")
)

type DynamicRecord struct {
	Record   Record
	ID       *int
	Disabled bool
}

func (r DynamicRecord) String() string {
	result := r.Record.String()

	if r.ID != nil {
		result += " ; ID:" + strconv.Itoa(*r.ID)
	}

	if r.Disabled {
		result = ";" + result
	}

	return result
}

func (r *DynamicRecord) UnmarshalText(text []byte) error {
	if bytes.Contains(text, []byte("\n")) {
		return ErrMultiLineDynamicRecord
	}

	text = bytes.TrimSpace(text)
	r.Disabled = false
	if text[0] == ';' {
		r.Disabled = true
		text = bytes.TrimSpace(bytes.TrimLeft(text, ";"))
	}

	xs := bytes.SplitN(text, []byte(";"), 2)
	if len(xs) != 2 {
		r.ID = nil
	} else {
		for _, x := range bytes.Split(bytes.TrimSpace(xs[1]), []byte(" ")) {
			kvs := bytes.SplitN(x, []byte(":"), 2)
			if len(kvs) != 2 {
				return ErrInvalidDynamicRecordFormat
			}

			switch string(bytes.TrimSpace(kvs[0])) {
			case "ID":
				id, err := strconv.Atoi(string(bytes.TrimSpace(kvs[1])))
				if err != nil {
					return ErrInvalidDynamicRecordFormat
				}
				r.ID = &id
			}
		}
	}

	rr, err := dns.NewRR(string(xs[0]))
	if err != nil {
		return err
	}

	r.Record, err = NewRecordFromRR(rr)
	if err != nil {
		return err
	}

	return nil
}

func (r DynamicRecord) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

type DynamicRecordSet []DynamicRecord

func (rs *DynamicRecordSet) UnmarshalText(text []byte) error {
	lines := bytes.Split(text, []byte("\n"))

	*rs = make([]DynamicRecord, 0, len(lines))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var r DynamicRecord
		if err := (&r).UnmarshalText(line); err != nil {
			if line[0] == ';' {
				continue
			} else {
				return err
			}
		}
		*rs = append(*rs, r)
	}

	return nil
}

func (rs DynamicRecordSet) MarshalText() ([]byte, error) {
	var err error

	bs := make([][]byte, len(rs)+1)  // add +1 element for put \n into last line
	for i, r := range rs {
		bs[i], err = r.MarshalText()
		if err != nil {
			return nil, err
		}
	}

	return bytes.Join(bs, []byte("\n")), nil
}

type DynamicResolver interface {
	Resolver

	UpdateAddresses(AddressesConfig) error
	GetAddresses() (AddressesConfig, error)

	UpdateCnames(CnamesConfig) error
	GetCnames() (CnamesConfig, error)

	UpdateTexts(TextsConfig) error
	GetTexts() (TextsConfig, error)

	UpdateServices(ServicesConfig) error
	GetServices() (ServicesConfig, error)

	//SetRecords(DynamicRecordSet) error
	//Records() (DynamicRecordSet, error)
}
