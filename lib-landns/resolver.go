package landns

import (
	"fmt"
	"io"
)

type Resolver interface {
	io.Closer

	Resolve(ResponseWriter, Request) error
	RecursionAvailable() bool
}

type ResolverSet []Resolver

func (rs ResolverSet) Resolve(resp ResponseWriter, req Request) error {
	for _, r := range rs {
		if err := r.Resolve(resp, req); err != nil {
			return err
		}
	}
	return nil
}

func (rs ResolverSet) RecursionAvailable() bool {
	for _, r := range rs {
		if r.RecursionAvailable() {
			return true
		}
	}
	return false
}

func (rs ResolverSet) Close() error {
	for _, r := range rs {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (rs ResolverSet) String() string {
	return fmt.Sprintf("ResolverSet%s", []Resolver(rs))
}

type AlternateResolver []Resolver

func (ar AlternateResolver) Resolve(resp ResponseWriter, req Request) error {
	resolved := false

	resp = ResponseWriterHook{
		Writer: resp,
		OnAdd: func(r Record) {
			resolved = true
		},
	}

	for _, r := range ar {
		if err := r.Resolve(resp, req); err != nil {
			return err
		}

		if resolved {
			return nil
		}
	}
	return nil
}

func (ar AlternateResolver) RecursionAvailable() bool {
	for _, r := range ar {
		if r.RecursionAvailable() {
			return true
		}
	}
	return false
}

func (ar AlternateResolver) Close() error {
	for _, r := range ar {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}
