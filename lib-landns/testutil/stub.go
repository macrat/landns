package testutil

import (
	"fmt"

	"github.com/macrat/landns/lib-landns"
)

type DummyResolver struct {
	Error     bool
	Recursion bool
}

func (dr DummyResolver) Resolve(w landns.ResponseWriter, r landns.Request) error {
	if dr.Error {
		return fmt.Errorf("test error")
	} else {
		return nil
	}
}

func (dr DummyResolver) RecursionAvailable() bool {
	return dr.Recursion
}

func (dr DummyResolver) Close() error {
	return nil
}

type DummyResponseWriter struct {
	Records       []landns.Record
	Authoritative bool
}

func NewDummyResponseWriter() *DummyResponseWriter {
	return &DummyResponseWriter{
		Records:       make([]landns.Record, 0, 10),
		Authoritative: true,
	}
}

func (rw *DummyResponseWriter) Add(r landns.Record) error {
	rw.Records = append(rw.Records, r)
	return nil
}

func (rw *DummyResponseWriter) IsAuthoritative() bool {
	return rw.Authoritative
}

func (rw *DummyResponseWriter) SetNoAuthoritative() {
	rw.Authoritative = false
}

type EmptyResponseWriter struct{}

func (rw EmptyResponseWriter) Add(r landns.Record) error {
	return nil
}

func (rw EmptyResponseWriter) IsAuthoritative() bool {
	return true
}

func (rw EmptyResponseWriter) SetNoAuthoritative() {
}
