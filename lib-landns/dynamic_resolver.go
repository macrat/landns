package landns

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var (
	ErrMultiLineDynamicRecord     = Error{Type: TypeArgumentError, Message: "DynamicRecord can't have multi line"}
	ErrInvalidDynamicRecordFormat = Error{Type: TypeArgumentError, Message: "DynamicRecord invalid format"}
	ErrNoSuchRecord               = Error{Type: TypeArgumentError, Message: "no such record"}
)

// DynamicRecord is the record information for DynamicResolver.
type DynamicRecord struct {
	Record   Record
	ID       *int
	Volatile bool
	Disabled bool
}

// NewDynamicRecord will parse record text and make new DynamicRecord.
func NewDynamicRecord(record string) (DynamicRecord, error) {
	var d DynamicRecord
	return d, d.UnmarshalText([]byte(record))
}

// String is get printable string.
func (r DynamicRecord) String() string {
	annotates := []string{}

	if r.ID != nil {
		annotates = append(annotates, "ID:"+strconv.Itoa(*r.ID))
	}

	if r.Volatile {
		annotates = append(annotates, "Volatile")
	}

	result := r.Record.String()

	if len(annotates) > 0 {
		result = strings.Join(append([]string{result, ";"}, annotates...), " ")
	}

	if r.Disabled {
		result = ";" + result
	}

	return result
}

func (r *DynamicRecord) unmarshalAnnotation(text []byte) error {
	r.ID = nil
	r.Volatile = false

	for _, x := range bytes.Split(bytes.TrimSpace(text), []byte(" ")) {
		kvs := bytes.SplitN(x, []byte(":"), 2)

		switch string(bytes.ToUpper(bytes.TrimSpace(kvs[0]))) {
		case "ID":
			if len(kvs) != 2 {
				return ErrInvalidDynamicRecordFormat
			}
			id, err := strconv.Atoi(string(bytes.TrimSpace(kvs[1])))
			if err != nil {
				return ErrInvalidDynamicRecordFormat
			}
			r.ID = &id
		case "VOLATILE":
			if len(kvs) != 1 {
				return ErrInvalidDynamicRecordFormat
			}
			r.Volatile = true
		}
	}

	return nil
}

// UnmarshalText is unmarshal DynamicRecord from text.
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
		r.unmarshalAnnotation(xs[1])
	}

	var err error
	r.Record, err = NewRecord(string(xs[0]))
	return err
}

// MarshalText is marshal DynamicRecord to text.
func (r DynamicRecord) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

// DynamicRecordSet is list of DynamicRecord
type DynamicRecordSet []DynamicRecord

// NewDynamicRecordSet will parse record text and make new DynamicRecordSet.
func NewDynamicRecordSet(records string) (DynamicRecordSet, error) {
	var d DynamicRecordSet
	return d, d.UnmarshalText([]byte(records))
}

func (rs *DynamicRecordSet) UnmarshalText(text []byte) error {
	lines := bytes.Split(text, []byte("\n"))

	*rs = make([]DynamicRecord, 0, len(lines))

	errors := ErrorSet{}

	for i, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var r DynamicRecord
		if err := (&r).UnmarshalText(line); err != nil {
			if line[0] == ';' {
				continue
			} else {
				errors = append(errors, newError(TypeArgumentError, nil, "line %d: invalid format: %s", i+1, string(line))) // unused original error because useless.
			}
		}
		*rs = append(*rs, r)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (rs DynamicRecordSet) MarshalText() ([]byte, error) {
	var err error

	bs := make([][]byte, len(rs)+1) // add +1 element for put \n into last line
	for i, r := range rs {
		bs[i], err = r.MarshalText()
		if err != nil {
			return nil, Error{TypeArgumentError, err, "failed to marshal record"}
		}
	}

	return bytes.Join(bs, []byte("\n")), nil
}

func (rs DynamicRecordSet) String() string {
	b, _ := rs.MarshalText()
	return string(b)
}

type ExpiredRecord struct {
	RR     dns.RR
	Expire time.Time
}

// NewExpiredRecord will parse record text and make new ExpiredRecord.
func NewExpiredRecord(record string) (ExpiredRecord, error) {
	var r ExpiredRecord
	return r, r.UnmarshalText([]byte(record))
}

// Record is Record getter.
func (r ExpiredRecord) Record() (Record, error) {
	if r.Expire.Unix() > 0 {
		ttl := math.Round(r.Expire.Sub(time.Now()).Seconds())
		if ttl < 0 {
			return nil, newError(TypeArgumentError, nil, "this record is already expired: %s", r.Expire)
		}

		r.RR.Header().Ttl = uint32(ttl)
	}

	return NewRecordFromRR(r.RR)
}

// String is get printable string.
func (r ExpiredRecord) String() string {
	text, _ := r.MarshalText()
	return string(text)
}

// UnmarshalText is unmarshal ExpiredRecord from text.
func (r *ExpiredRecord) UnmarshalText(text []byte) error {
	if bytes.Contains(text, []byte("\n")) {
		return ErrMultiLineDynamicRecord
	}
	text = bytes.TrimSpace(text)

	r.Expire = time.Unix(0, 0)

	xs := bytes.SplitN(text, []byte(";"), 2)
	if len(xs) == 2 {
		i, err := strconv.ParseInt(string(bytes.TrimSpace(xs[1])), 10, 64)
		if err != nil {
			return Error{TypeInternalError, err, "failed to parse record"}
		}
		r.Expire = time.Unix(i, 0)

		if r.Expire.Before(time.Now()) {
			return newError(TypeArgumentError, nil, "failed to parse record: expire can't be past time: %s", r.Expire)
		}
	}

	var err error
	r.RR, err = dns.NewRR(string(xs[0]))
	if err != nil {
		return Error{TypeInternalError, err, "failed to parse record"}
	}

	return nil
}

// MarshalText is marshal ExpiredRecord to text.
func (r ExpiredRecord) MarshalText() ([]byte, error) {
	rec, err := r.Record()
	if err != nil {
		return nil, err
	}

	if r.Expire.Unix() > 0 {
		return []byte(fmt.Sprintf("%s ; %d", rec, r.Expire.Unix())), nil
	}

	return []byte(rec.String()), nil
}

type DynamicResolver interface {
	Resolver

	SetRecords(DynamicRecordSet) error
	Records() (DynamicRecordSet, error)
	SearchRecords(Domain) (DynamicRecordSet, error)
	GlobRecords(string) (DynamicRecordSet, error)
	GetRecord(int) (DynamicRecordSet, error)
	RemoveRecord(int) error
}
