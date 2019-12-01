package landns

import (
	"fmt"
)

type Resolver interface {
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

func (rs ResolverSet) String() string {
	return fmt.Sprintf("ResolverSet%s", []Resolver(rs))
}

type AlternateResolver []Resolver

func (ar AlternateResolver) Resolve(resp ResponseWriter, req Request) error {
	resolved := false

	respWrap := NewResponseCallback(func(r Record) error {
		resolved = true
		return resp.Add(r)
	})

	for _, r := range ar {
		if err := r.Resolve(respWrap, req); err != nil {
			return err
		}

		if resolved {
			if !respWrap.Authoritative {
				resp.SetNoAuthoritative()
			}
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
