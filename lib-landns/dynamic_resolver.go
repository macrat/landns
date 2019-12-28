package landns

import (
	"bytes"
	"strconv"
	"strings"
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
	Disabled bool
}

// NewDynamicRecord will parse record text and make new DynamicRecord.
func NewDynamicRecord(record string) (DynamicRecord, error) {
	var d DynamicRecord
	return d, d.UnmarshalText([]byte(record))
}

// String is get printable string.
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

func (r *DynamicRecord) unmarshalAnnotation(text []byte) error {
	r.ID = nil

	for _, x := range bytes.Split(bytes.TrimSpace(text), []byte(" ")) {
		kvs := bytes.SplitN(x, []byte(":"), 2)

		switch strings.ToUpper(string(bytes.TrimSpace(kvs[0]))) {
		case "ID":
			if len(kvs) != 2 {
				return ErrInvalidDynamicRecordFormat
			}
			id, err := strconv.Atoi(string(bytes.TrimSpace(kvs[1])))
			if err != nil {
				return ErrInvalidDynamicRecordFormat
			}
			r.ID = &id
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

type DynamicResolver interface {
	Resolver

	SetRecords(DynamicRecordSet) error
	Records() (DynamicRecordSet, error)
	SearchRecords(Domain) (DynamicRecordSet, error)
	GlobRecords(string) (DynamicRecordSet, error)
	GetRecord(int) (DynamicRecordSet, error)
	RemoveRecord(int) error
}
