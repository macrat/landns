package landns

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrMultiLineDynamicRecord     = fmt.Errorf("DynamicRecord can't have multi line")
	ErrInvalidDynamicRecordFormat = fmt.Errorf("DynamicRecord invalid format")
	ErrNoSuchRecord               = fmt.Errorf("no such record")
)

type InvalidRecordError struct {
	Line int
	Text string
}

func (e InvalidRecordError) Error() string {
	return fmt.Sprintf("line %d: invalid format: %s", e.Line, e.Text)
}

type ErrorSet []error

func (e ErrorSet) Error() string {
	xs := make([]string, len(e))
	for i, x := range e {
		xs[i] = x.Error()
	}
	return strings.Join(xs, "\n")
}

type DynamicRecord struct {
	Record   Record
	ID       *int
	Disabled bool
}

func NewDynamicRecord(record string) (DynamicRecord, error) {
	var d DynamicRecord
	return d, d.UnmarshalText([]byte(record))
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

	var err error
	r.Record, err = NewRecord(string(xs[0]))
	return err
}

func (r DynamicRecord) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

type DynamicRecordSet []DynamicRecord

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
				errors = append(errors, InvalidRecordError{i + 1, string(line)})
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
			return nil, err
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
