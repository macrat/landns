package landns

import (
	"fmt"
)

type Resolver interface {
	Resolve(ResponseWriter, Request) error
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

func (rs ResolverSet) String() string {
	return fmt.Sprintf("ResolverSet%s", []Resolver(rs))
}
