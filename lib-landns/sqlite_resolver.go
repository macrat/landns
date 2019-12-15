package landns

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/miekg/dns"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// SqliteResolver is one implements of DynamicResolver using Sqlite3.
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

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		qtype TEXT,
		record TEXT UNIQUE
	)`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS record_name ON records (name, qtype)`)
	if err != nil {
		return nil, err
	}

	return &SqliteResolver{path, db, metrics}, nil
}

func (sr *SqliteResolver) String() string {
	return fmt.Sprintf("SqliteResolver[%s]", sr.path)
}

func insertRecord(search, ins *sql.Stmt, r DynamicRecord) error {
	result, err := search.Query(r.Record.String())
	if err != nil {
		return err
	}
	defer result.Close()

	if result.Next() {
		return nil
	}

	_, err = ins.Exec(r.Record.GetName(), QtypeToString(r.Record.GetQtype()), r.Record.String())
	if err != nil {
		return err
	}

	if r.Record.GetQtype() == dns.TypeA || r.Record.GetQtype() == dns.TypeAAAA {
		reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
		if err != nil {
			return err
		}
		ins.Exec(reverse, "PTR", fmt.Sprintf("%s %d IN PTR %s", reverse, r.Record.GetTTL(), r.Record.GetName()))
	}

	return nil
}

func dropRecord(withID, withoutID *sql.Stmt, r DynamicRecord) error {
	if r.ID == nil {
		_, err := withoutID.Exec(r.Record.String())
		if err != nil {
			return err
		}
	} else {
		_, err := withID.Exec(*r.ID, r.Record.String())
		if err != nil {
			return err
		}
	}

	if r.Record.GetQtype() == dns.TypeA || r.Record.GetQtype() == dns.TypeAAAA {
		reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
		if err != nil {
			return err
		}
		withoutID.Exec(fmt.Sprintf("%s %d IN PTR %s", reverse, r.Record.GetTTL(), r.Record.GetName()))
	}

	return nil
}

func (sr *SqliteResolver) SetRecords(rs DynamicRecordSet) error {
	tx, err := sr.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	dropWithID, err := tx.Prepare(`DELETE FROM records WHERE id = ? AND record = ?`)
	if err != nil {
		return err
	}
	defer dropWithID.Close()

	dropWithoutID, err := tx.Prepare(`DELETE FROM records WHERE record = ?`)
	if err != nil {
		return err
	}
	defer dropWithoutID.Close()

	search, err := tx.Prepare(`SELECT 1 FROM records WHERE record = ? LIMIT 1`)
	if err != nil {
		return err
	}
	defer search.Close()

	ins, err := tx.Prepare(`INSERT INTO records (name, qtype, record) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer ins.Close()

	for _, r := range rs {
		if r.Disabled {
			if err := dropRecord(dropWithID, dropWithoutID, r); err != nil {
				return err
			}
		} else {
			if err := insertRecord(search, ins, r); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func scanRecords(rows *sql.Rows) (DynamicRecordSet, error) {
	var text string
	var result DynamicRecordSet

	for rows.Next() {
		var dr DynamicRecord

		if err := rows.Scan(&dr.ID, &text); err != nil {
			return DynamicRecordSet{}, err
		}

		var err error
		dr.Record, err = NewRecord(text)
		if err != nil {
			return DynamicRecordSet{}, err
		}

		result = append(result, dr)
	}

	return result, nil
}

func (sr *SqliteResolver) Records() (DynamicRecordSet, error) {
	rows, err := sr.db.Query(`SELECT id, record FROM records ORDER BY id`)
	if err != nil {
		return DynamicRecordSet{}, err
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) SearchRecords(suffix Domain) (DynamicRecordSet, error) {
	suf := suffix.String()
	for _, rep := range []struct {
		From string
		To   string
	}{
		{`\`, `\\`},
		{`%`, `\%`},
		{`_`, `\_`},
	} {
		suf = strings.ReplaceAll(suf, rep.From, rep.To)
	}

	rows, err := sr.db.Query(`SELECT id, record FROM records WHERE name = ? OR name LIKE ? ESCAPE '\' ORDER BY id`, suf, "%."+suf)
	if err != nil {
		return DynamicRecordSet{}, err
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) GlobRecords(pattern string) (DynamicRecordSet, error) {
	for _, rep := range []struct {
		From string
		To   string
	}{
		{`\`, `\\`},
		{`%`, `\%`},
		{`_`, `\_`},
		{`*`, `%`},
	} {
		pattern = strings.ReplaceAll(pattern, rep.From, rep.To)
	}

	rows, err := sr.db.Query(`SELECT id, record FROM records WHERE name LIKE ? ESCAPE '\' ORDER BY id`, pattern)
	if err != nil {
		return DynamicRecordSet{}, err
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) GetRecord(id int) (DynamicRecordSet, error) {
	rows, err := sr.db.Query(`SELECT id, record FROM records WHERE id = ?`, id)
	if err != nil {
		return DynamicRecordSet{}, err
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) RemoveRecord(id int) error {
	result, err := sr.db.Exec(`DELETE FROM records WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNoSuchRecord
	}
	return nil
}

func (sr *SqliteResolver) Resolve(w ResponseWriter, r Request) error {
	rows, err := sr.db.Query(`SELECT record FROM records WHERE name = ? AND qtype = ?`, r.Name, r.QtypeString())
	if err != nil {
		return err
	}
	defer rows.Close()

	var text string

	for rows.Next() {
		if err := rows.Scan(&text); err != nil {
			return err
		}

		record, err := NewRecord(text)
		if err != nil {
			return err
		}

		if err := w.Add(record); err != nil {
			return err
		}
	}

	return nil
}

func (sr *SqliteResolver) RecursionAvailable() bool {
	return false
}

func (sr *SqliteResolver) Close() error {
	return sr.db.Close()
}
