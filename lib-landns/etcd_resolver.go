package landns

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	"go.etcd.io/etcd/clientv3"
)

// EtcdResolver is one implements of DynamicResolver using etcd.
type EtcdResolver struct {
	client *clientv3.Client

	Timeout time.Duration
	Prefix  string
}

// NewEtcdResolver is constructor of EtcdResolver.
func NewEtcdResolver(endpoints []string, prefix string, timeout time.Duration, metrics *Metrics) (*EtcdResolver, error) {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: timeout,
	})
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to connect etcd"}
	}

	return &EtcdResolver{
		client:  c,
		Timeout: timeout,
		Prefix:  prefix,
	}, nil
}

// String is description string getter.
func (er *EtcdResolver) String() string {
	return fmt.Sprintf("EtcdResolver%s", er.client.Endpoints())
}

func (er *EtcdResolver) makeContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), er.Timeout)
}

func (er *EtcdResolver) makeKey(ctx context.Context, r DynamicRecord) (DynamicRecord, string, error) {
	if r.ID == nil {
		resp, err := er.client.Get(ctx, er.Prefix+"/lastID")
		if err != nil {
			return r, "", Error{TypeExternalError, err, "failed to get last ID"}
		}

		id := 0
		if len(resp.Kvs) > 0 {
			id, err = strconv.Atoi(string(resp.Kvs[0].Value))
			if err != nil {
				return r, "", Error{TypeInternalError, err, "failed to parse last ID"}
			}
		}
		id++
		r.ID = &id

		er.client.Put(ctx, er.Prefix+"/lastID", strconv.Itoa(*r.ID))
	}

	return er.findKey(ctx, r)
}

func (er *EtcdResolver) findKey(ctx context.Context, r DynamicRecord) (DynamicRecord, string, error) {
	if r.ID != nil {
		return r, fmt.Sprintf("%s/records%s/%d", er.Prefix, r.Record.GetName().ToPath(), *r.ID), nil
	}

	resp, err := er.client.Get(ctx, er.Prefix+"/records"+r.Record.GetName().ToPath()+"/", clientv3.WithPrefix())
	if err != nil {
		return DynamicRecord{}, "", Error{TypeExternalError, err, "failed to get records"}
	}
	var r2 DynamicRecord
	for _, x := range resp.Kvs {
		if err := r2.UnmarshalText(x.Value); err != nil {
			return DynamicRecord{}, "", Error{TypeInternalError, err, "failed to parse records"}
		}

		if r.Record.String() == r2.Record.String() {
			ks := strings.Split(string(x.Key), "/")
			id, err := strconv.Atoi(ks[len(ks)-1])
			if err != nil {
				return DynamicRecord{}, "", Error{TypeInternalError, err, "failed to parse record ID"}
			}
			r2.ID = &id
			return er.makeKey(ctx, r2)
		}
	}

	return DynamicRecord{}, "", nil
}

func (er *EtcdResolver) getIDbyKey(key string) (int, error) {
	ks := strings.Split(key, "/")
	return strconv.Atoi(ks[len(ks)-1])
}

func (er *EtcdResolver) readResponses(resp *clientv3.GetResponse) (DynamicRecordSet, error) {
	rs := make(DynamicRecordSet, len(resp.Kvs))

	for i, r := range resp.Kvs {
		if err := rs[i].UnmarshalText(r.Value); err != nil {
			return nil, Error{TypeInternalError, err, "faield to parse records"}
		}

		id, err := er.getIDbyKey(string(r.Key))
		if err != nil {
			return nil, err
		}
		rs[i].ID = &id
	}

	return rs, nil
}

func (er *EtcdResolver) findSameRecord(ctx context.Context, r DynamicRecord) (bool, error) {
	resp, err := er.client.Get(ctx, er.Prefix+"/records"+r.Record.GetName().ToPath(), clientv3.WithPrefix())
	if err != nil {
		return false, Error{TypeExternalError, err, "failed to get records"}
	}
	for _, r2 := range resp.Kvs {
		if r.String() == string(r2.Value) {
			return true, nil
		}
	}
	return false, nil
}

func (er *EtcdResolver) dropRecord(ctx context.Context, r DynamicRecord) error {
	_, key, err := er.findKey(ctx, r)
	if err != nil {
		return err
	}

	if key != "" {
		if _, err = er.client.Delete(ctx, key); err != nil {
			return Error{TypeExternalError, err, "failed to delete record"}
		}
	}

	if r.Record.GetQtype() != dns.TypeA && r.Record.GetQtype() != dns.TypeAAAA {
		return nil
	}

	reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
	if err != nil {
		return Error{TypeExternalError, err, "failed to make reverse address"}
	}
	_, key, err = er.findKey(ctx, DynamicRecord{
		Record: PtrRecord{
			Name:   Domain(reverse),
			TTL:    r.Record.GetTTL(),
			Domain: r.Record.GetName(),
		},
	})
	if err != nil {
		return err
	} else if key == "" {
		return nil
	}
	if _, err := er.client.Delete(ctx, key); err != nil {
		return Error{TypeExternalError, err, "failed to delete record"}
	}

	return nil
}

func (er *EtcdResolver) insertRecord(ctx context.Context, r DynamicRecord) error {
	if found, err := er.findSameRecord(ctx, r); err != nil {
		return err
	} else if found {
		return nil
	}

	r, key, err := er.makeKey(ctx, r)
	if err != nil {
		return err
	}

	if _, err := er.client.Put(ctx, key, r.Record.String()); err != nil {
		return Error{TypeExternalError, err, "failed to put record"}
	}

	if r.Record.GetQtype() == dns.TypeA || r.Record.GetQtype() == dns.TypeAAAA {
		reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
		if err != nil {
			return Error{TypeExternalError, err, "failed to make reverse address"}
		}
		r, key, err = er.makeKey(ctx, DynamicRecord{
			Record: PtrRecord{
				Name:   Domain(reverse),
				TTL:    r.Record.GetTTL(),
				Domain: r.Record.GetName(),
			},
		})
		if err != nil {
			return err
		}
		if _, err := er.client.Put(ctx, key, r.Record.String()); err != nil {
			return Error{TypeExternalError, err, "failed to put record"}
		}
	}

	return nil
}

// SetRecords is DynamicRecord setter.
func (er *EtcdResolver) SetRecords(rs DynamicRecordSet) error {
	ctx, cancel := er.makeContext()
	defer cancel()

	for _, r := range rs {
		var err error

		if r.Disabled {
			err = er.dropRecord(ctx, r)
		} else {
			err = er.insertRecord(ctx, r)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// Records is DynamicRecord getter.
func (er *EtcdResolver) Records() (DynamicRecordSet, error) {
	ctx, cancel := er.makeContext()
	defer cancel()

	resp, err := er.client.Get(ctx, er.Prefix+"/records/", clientv3.WithPrefix())
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to get records"}
	}

	return er.readResponses(resp)
}

// SearchRecords is search records by domain prefix.
func (er *EtcdResolver) SearchRecords(d Domain) (DynamicRecordSet, error) {
	ctx, cancel := er.makeContext()
	defer cancel()

	resp, err := er.client.Get(ctx, er.Prefix+"/records"+d.ToPath(), clientv3.WithPrefix())
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to get records"}
	}

	return er.readResponses(resp)
}

func compileGlob(glob string) (func(string) bool, error) {
	for _, x := range []struct {
		From string
		To   string
	}{
		{`\`, `\\`},
		{`.`, `\.`},
		{`+`, `\+`},
		{`[`, `\[`},
		{`]`, `\]`},
		{`(`, `\(`},
		{`)`, `\)`},
		{`^`, `\^`},
		{`$`, `\$`},
		{`*`, `.*`},
	} {
		glob = strings.ReplaceAll(glob, x.From, x.To)
	}

	re, err := regexp.Compile("^" + glob + "$")
	if err != nil {
		return nil, Error{TypeInternalError, err, "failed to parse glob"}
	}

	return re.MatchString, nil
}

// GlobRecords is search records by glob string.
func (er *EtcdResolver) GlobRecords(glob string) (DynamicRecordSet, error) {
	check, err := compileGlob(glob)
	if err != nil {
		return nil, err
	}

	rs, err := er.Records()
	if err != nil {
		return nil, err
	}

	result := make(DynamicRecordSet, 0, len(rs))
	for _, r := range rs {
		if check(r.Record.GetName().String()) {
			result = append(result, r)
		}
	}
	return result, nil
}

// GetRecord is get record by id.
func (er *EtcdResolver) GetRecord(id int) (DynamicRecordSet, error) {
	rs, err := er.Records()
	if err != nil {
		return nil, err
	}

	for _, r := range rs {
		if *r.ID == id {
			return DynamicRecordSet{r}, nil
		}
	}
	return DynamicRecordSet{}, nil
}

// RemoveRecord is remove record by id.
func (er *EtcdResolver) RemoveRecord(id int) error {
	ctx, cancel := er.makeContext()
	defer cancel()

	rs, err := er.Records()
	if err != nil {
		return err
	}

	for _, r := range rs {
		if *r.ID == id {
			_, key, err := er.findKey(ctx, r)
			if err != nil {
				return err
			}
			_, err = er.client.Delete(ctx, key)
			if err != nil {
				return Error{TypeExternalError, err, "failed to delete record"}
			}
		}
	}
	return nil
}

// RecursionAvailable is always returns `false`.
func (er *EtcdResolver) RecursionAvailable() bool {
	return false
}

// Close is disconnector from etcd server.
func (er *EtcdResolver) Close() error {
	err := er.client.Close()
	if err != nil {
		return Error{TypeExternalError, err, "failed to close etcd connection"}
	}
	return nil
}

// Resolve is resolver using etcd.
func (er *EtcdResolver) Resolve(w ResponseWriter, r Request) error {
	name := Domain(r.Name)

	rs, err := er.SearchRecords(name)
	if err != nil {
		return err
	}

	for _, rec := range rs {
		if rec.Record.GetName() == name && rec.Record.GetQtype() == r.Qtype {
			if err := w.Add(rec.Record); err != nil {
				return err
			}
		}
	}

	return nil
}
