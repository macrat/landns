package landns

import (
	"database/sql"
	"fmt"
	"net"

	"github.com/miekg/dns"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

func iterAddresses(rows *sql.Rows, callback func(AddressRecord)) error {
	for rows.Next() {
		var name string
		var addr string
		var ttl uint32

		if err := rows.Scan(&name, &addr, &ttl); err != nil {
			return err
		}

		callback(AddressRecord{Name: Domain(name), TTL: ttl, Address: net.ParseIP(addr)})
	}
	return nil
}

func iterCnames(rows *sql.Rows, callback func(CnameRecord)) error {
	for rows.Next() {
		var name string
		var target string
		var ttl uint32

		if err := rows.Scan(&name, &target, &ttl); err != nil {
			return err
		}

		callback(CnameRecord{Name: Domain(name), TTL: ttl, Target: Domain(target)})
	}
	return nil
}

func iterTexts(rows *sql.Rows, callback func(TxtRecord)) error {
	for rows.Next() {
		var name string
		var text string
		var ttl uint32

		if err := rows.Scan(&name, &text, &ttl); err != nil {
			return err
		}

		callback(TxtRecord{Name: Domain(name), TTL: ttl, Text: text})
	}
	return nil
}

func iterServices(rows *sql.Rows, callback func(SrvRecord)) error {
	for rows.Next() {
		var name string
		var priority uint16
		var weight uint16
		var port uint16
		var target string
		var ttl uint32

		if err := rows.Scan(&name, &priority, &weight, &port, &target, &ttl); err != nil {
			return err
		}

		callback(SrvRecord{
			Name:     Domain(name),
			TTL:      ttl,
			Priority: priority,
			Weight:   weight,
			Port:     port,
			Target:   Domain(target),
		})
	}
	return nil
}

type SqliteResolver struct {
	path    string
	db      *sql.DB
	metrics *Metrics
}

func NewSqliteResolver(path string, metrics *Metrics) (*SqliteResolver, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS addresses (
		name TEXT,
		address TEXT,
		reverse_address TEXT NOT NULL,
		ttl INTEGER NOT NULL CHECK(0 <= ttl),
		PRIMARY KEY (name, address)
	)`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cnames (
		name TEXT,
		target TEXT,
		ttl INTEGER NOT NULL CHECK(0 <= ttl),
		PRIMARY KEY (name, target)
	)`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS texts (
		name TEXT,
		text TEXT,
		ttl INTEGER NOT NULL CHECK(0 <= ttl),
		PRIMARY KEY (name, text)
	)`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS services (
		name TEXT,
		priority INTEGER NOT NULL CHECK(0 <= priority AND priority <= 65535),
		weight INTEGER NOT NULL CHECK(0 <= weight AND weight <= 65535),
		port INTEGER CHECK(0 <= port AND port <= 65535),
		target TEXT,
		ttl INTEGER NOT NULL CHECK(0 <= ttl),
		PRIMARY KEY (name, port, target)
	)`)
	if err != nil {
		return nil, err
	}

	return &SqliteResolver{path, db, metrics}, nil
}

func (r *SqliteResolver) Close() {
	r.db.Close()
}

func (r *SqliteResolver) String() string {
	return fmt.Sprintf("SqliteResolver[%s]", r.path)
}

func (r *SqliteResolver) UpdateAddresses(config AddressesConfig) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	drop, err := tx.Prepare(`DELETE FROM addresses WHERE name = ?`)
	if err != nil {
		return err
	}
	defer drop.Close()

	ins, err := tx.Prepare(`INSERT INTO addresses VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer ins.Close()

	for name, list := range config {
		if _, err = drop.Exec(name.String()); err != nil {
			return err
		}

		for _, r := range list {
			rf := r.Normalized()

			reverse, err := dns.ReverseAddr(rf.Address.String())
			if err != nil {
				return err
			}

			if _, err = ins.Exec(name.String(), rf.Address.String(), reverse, rf.TTL); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *SqliteResolver) GetAddresses() (AddressesConfig, error) {
	rows, err := r.db.Query(`SELECT name, address, ttl FROM addresses`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := make(AddressesConfig)

	return resp, iterAddresses(rows, func(r AddressRecord) {
		resp.RegisterRecord(r)
	})
}

func (r *SqliteResolver) UpdateCnames(config CnamesConfig) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	drop, err := tx.Prepare(`DELETE FROM cnames WHERE name = ?`)
	if err != nil {
		return err
	}
	defer drop.Close()

	ins, err := tx.Prepare(`INSERT INTO cnames VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer ins.Close()

	for name, list := range config {
		if _, err = drop.Exec(name.String()); err != nil {
			return err
		}

		for _, r := range list {
			rf := r.Normalized()
			if _, err = ins.Exec(name.String(), rf.Target.String(), rf.TTL); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *SqliteResolver) GetCnames() (CnamesConfig, error) {
	rows, err := r.db.Query(`SELECT name, target, ttl FROM cnames`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := make(CnamesConfig)

	return resp, iterCnames(rows, func(r CnameRecord) {
		resp.RegisterRecord(r)
	})
}

func (r *SqliteResolver) UpdateTexts(config TextsConfig) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	drop, err := tx.Prepare(`DELETE FROM texts WHERE name = ?`)
	if err != nil {
		return err
	}
	defer drop.Close()

	ins, err := tx.Prepare(`INSERT INTO texts VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer ins.Close()

	for name, list := range config {
		if _, err = drop.Exec(name.String()); err != nil {
			return err
		}

		for _, r := range list {
			rf := r.Normalized()
			if _, err = ins.Exec(name.String(), rf.Text, rf.TTL); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *SqliteResolver) GetTexts() (TextsConfig, error) {
	rows, err := r.db.Query(`SELECT name, text, ttl FROM texts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := make(TextsConfig)

	return resp, iterTexts(rows, func(r TxtRecord) {
		resp.RegisterRecord(r)
	})
}

func (r *SqliteResolver) UpdateServices(config ServicesConfig) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	drop, err := tx.Prepare(`DELETE FROM services WHERE name = ?`)
	if err != nil {
		return err
	}
	defer drop.Close()

	ins, err := tx.Prepare(`INSERT INTO services VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer ins.Close()

	for name, list := range config {
		if _, err = drop.Exec(name.String()); err != nil {
			return err
		}

		for _, r := range list {
			rf := r.Normalized()
			if _, err = ins.Exec(name.String(), rf.Priority, rf.Weight, rf.Port, rf.Target.String(), rf.TTL); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *SqliteResolver) GetServices() (ServicesConfig, error) {
	rows, err := r.db.Query(`SELECT name, priority, weight, port, target, ttl FROM services`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := make(ServicesConfig)

	return resp, iterServices(rows, func(r SrvRecord) {
		resp.RegisterRecord(r)
	})
}

func (r *SqliteResolver) ResolveAddresses(name string) (resp []AddressRecord, err error) {
	rows, err := r.db.Query(`SELECT name, address, ttl FROM addresses WHERE name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return resp, iterAddresses(rows, func(r AddressRecord) {
		resp = append(resp, r)
	})
}

func (r *SqliteResolver) ResolveA(name string) (resp []Record, err error) {
	ipresp, err := r.ResolveAddresses(name)
	if err != nil {
		return nil, err
	}

	for _, r := range ipresp {
		if r.IsV4() {
			resp = append(resp, r)
		}
	}
	return
}

func (r *SqliteResolver) ResolveAAAA(name string) (resp []Record, err error) {
	ipresp, err := r.ResolveAddresses(name)
	if err != nil {
		return nil, err
	}

	for _, r := range ipresp {
		if !r.IsV4() {
			resp = append(resp, r)
		}
	}
	return
}

func (r *SqliteResolver) ResolvePTR(addr string) (resp []Record, err error) {
	rows, err := r.db.Query(`SELECT name, address, ttl FROM addresses WHERE reverse_address = ?`, addr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return resp, iterAddresses(rows, func(r AddressRecord) {
		resp = append(resp, PtrRecord{
			Name:   Domain(addr),
			TTL:    r.TTL,
			Domain: r.Name,
		})
	})
}

func (r *SqliteResolver) ResolveCNAME(name string) (resp []Record, err error) {
	rows, err := r.db.Query(`SELECT name, target, ttl FROM cnames WHERE name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return resp, iterCnames(rows, func(r CnameRecord) {
		resp = append(resp, r)
	})
}

func (r *SqliteResolver) ResolveTXT(name string) (resp []Record, err error) {
	rows, err := r.db.Query(`SELECT name, text, ttl FROM texts WHERE name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return resp, iterTexts(rows, func(r TxtRecord) {
		resp = append(resp, r)
	})
}

func (r *SqliteResolver) ResolveSRV(name string) (resp []Record, err error) {
	rows, err := r.db.Query(`SELECT name, priority, weight, port, target, ttl FROM services WHERE name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return resp, iterServices(rows, func(r SrvRecord) {
		resp = append(resp, r)
	})
}

func (r *SqliteResolver) Resolve(resp ResponseWriter, req Request) error {
	var rs []Record
	var err error

	switch req.Qtype {
	case dns.TypeA:
		rs, err = r.ResolveA(req.Name)
	case dns.TypeAAAA:
		rs, err = r.ResolveAAAA(req.Name)
	case dns.TypePTR:
		rs, err = r.ResolvePTR(req.Name)
	case dns.TypeCNAME:
		rs, err = r.ResolveCNAME(req.Name)
	case dns.TypeTXT:
		rs, err = r.ResolveTXT(req.Name)
	case dns.TypeSRV:
		rs, err = r.ResolveSRV(req.Name)
	}
	for _, r := range rs {
		resp.Add(r)
	}
	return err
}

func (r *SqliteResolver) RecursionAvailable() bool {
	return false
}
