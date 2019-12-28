package landns

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/macrat/landns/lib-landns/logger"
	"github.com/miekg/dns"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// SqliteResolver is one implements of DynamicResolver using Sqlite3.
type SqliteResolver struct {
	mutex   sync.Mutex
	path    string
	db      *sql.DB
	metrics *Metrics
	closer  chan struct{}
}

func NewSqliteResolver(path string, metrics *Metrics) (*SqliteResolver, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to open SQlite database"}
	}

	sr := &SqliteResolver{
		path:    path,
		db:      db,
		metrics: metrics,
		closer:  make(chan struct{}),
	}

	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		qtype TEXT NOT NULL,
		expire INTEGER NOT NULL,
		record TEXT UNIQUE
	)`)
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to create table"}
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS record_name ON records (name, qtype)`)
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to create index"}
	}

	go sr.manageExpire(5 * time.Second)

	return sr, nil
}

func (sr *SqliteResolver) manageExpire(interval time.Duration) {
	sr.mutex.Lock()
	stmt, err := sr.db.Prepare(`
		DELETE FROM records
		WHERE expire > 0 AND expire < strftime('%s', CURRENT_TIMESTAMP)
	`)
	sr.mutex.Unlock()
	if err != nil && err.Error() != "sql: database is closed" {
		panic(err.Error())
	}

	tick := time.Tick(interval)

	for {
		select {
		case <-tick:
			sr.mutex.Lock()
			_, err := stmt.Exec()
			sr.mutex.Unlock()

			if err != nil && err.Error() != "sql: database is closed" {
				logger.Error("failed to delete expired records", logger.Fields{"reason": err})
			}
		case <-sr.closer:
			return
		}
	}
}

func (sr *SqliteResolver) String() string {
	return fmt.Sprintf("SqliteResolver[%s]", sr.path)
}

func insertRecord(search, ins *sql.Stmt, r DynamicRecord) error {
	result, err := search.Query(r.Record.String())
	if err != nil {
		return Error{TypeExternalError, err, "failed to search exists record"}
	}
	defer result.Close()

	if result.Next() {
		return nil
	}

	var expire int64
	if r.Volatile {
		expire = time.Now().Add(time.Duration(r.Record.GetTTL()) * time.Second).Unix()
	}

	_, err = ins.Exec(r.Record.GetName(), QtypeToString(r.Record.GetQtype()), expire, r.Record.String())
	if err != nil {
		return Error{TypeExternalError, err, "failed to insert record"}
	}

	if r.Record.GetQtype() == dns.TypeA || r.Record.GetQtype() == dns.TypeAAAA {
		reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
		if err != nil {
			return newError(TypeArgumentError, err, "failed to convert to reverse address: %s", r.Record.(AddressRecord).Address)
		}
		_, err = ins.Exec(reverse, "PTR", expire, fmt.Sprintf("%s %d IN PTR %s", reverse, r.Record.GetTTL(), r.Record.GetName()))
		if err != nil {
			return Error{TypeExternalError, err, "failed to insert reverse record"}
		}
	}

	return nil
}

func dropRecord(withID, withoutID *sql.Stmt, r DynamicRecord) error {
	if r.ID == nil {
		_, err := withoutID.Exec(r.Record.String())
		if err != nil {
			return Error{TypeExternalError, err, "failed to drop record"}
		}
	} else {
		_, err := withID.Exec(*r.ID, r.Record.String())
		if err != nil {
			return Error{TypeExternalError, err, "failed to drop record"}
		}
	}

	if r.Record.GetQtype() == dns.TypeA || r.Record.GetQtype() == dns.TypeAAAA {
		reverse, err := dns.ReverseAddr(r.Record.(AddressRecord).Address.String())
		if err != nil {
			return newError(TypeArgumentError, err, "failed to convert to reverse address: %s", r.Record.(AddressRecord).Address)
		}
		_, err = withoutID.Exec(fmt.Sprintf("%s %d IN PTR %s", reverse, r.Record.GetTTL(), r.Record.GetName()))
		if err != nil {
			return Error{TypeExternalError, err, "failed to drop reverse record"}
		}
	}

	return nil
}

func (sr *SqliteResolver) SetRecords(rs DynamicRecordSet) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	tx, err := sr.db.Begin()
	if err != nil {
		return Error{TypeExternalError, err, "failed to begin transaction"}
	}

	dropWithID, err := tx.Prepare(`DELETE FROM records WHERE id = ? AND record = ?`)
	if err != nil {
		tx.Rollback()
		return Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer dropWithID.Close()

	dropWithoutID, err := tx.Prepare(`DELETE FROM records WHERE record = ?`)
	if err != nil {
		tx.Rollback()
		return Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer dropWithoutID.Close()

	search, err := tx.Prepare(`
		SELECT 1 FROM records
		WHERE record = ?
		AND (expire = 0 OR expire > strftime('%s', CURRENT_TIMESTAMP))
		LIMIT 1
	`)
	if err != nil {
		tx.Rollback()
		return Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer search.Close()

	ins, err := tx.Prepare(`INSERT INTO records (name, qtype, expire, record) VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer ins.Close()

	for _, r := range rs {
		if r.Disabled {
			if err := dropRecord(dropWithID, dropWithoutID, r); err != nil {
				tx.Rollback()
				return err
			}
		} else {
			if err := insertRecord(search, ins, r); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func scanRecords(rows *sql.Rows) (DynamicRecordSet, error) {
	var expire int64
	var text string
	var result DynamicRecordSet

	for rows.Next() {
		var dr DynamicRecord

		if err := rows.Scan(&dr.ID, &expire, &text); err != nil {
			return DynamicRecordSet{}, Error{TypeExternalError, err, "failed to scan record row"}
		}

		var err error
		if expire != 0 {
			dr.Record, err = NewRecordWithExpire(text, time.Unix(expire, 0))
			dr.Volatile = true
		} else {
			dr.Record, err = NewRecord(text)
		}
		if err != nil {
			return DynamicRecordSet{}, err
		}

		result = append(result, dr)
	}

	return result, nil
}

func (sr *SqliteResolver) Records() (DynamicRecordSet, error) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	rows, err := sr.db.Query(`
		SELECT id, expire, record FROM records
		WHERE (expire = 0 OR expire > strftime('%s', CURRENT_TIMESTAMP))
		ORDER BY id
	`)
	if err != nil {
		return DynamicRecordSet{}, Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) SearchRecords(suffix Domain) (DynamicRecordSet, error) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

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

	rows, err := sr.db.Query(`
		SELECT id, expire, record FROM records
		WHERE (name = ? OR name LIKE ? ESCAPE '\')
		AND (expire = 0 OR expire > strftime('%s', CURRENT_TIMESTAMP))
		ORDER BY id
	`, suf, "%."+suf)
	if err != nil {
		return DynamicRecordSet{}, Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) GlobRecords(pattern string) (DynamicRecordSet, error) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

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

	rows, err := sr.db.Query(`
		SELECT id, expire, record FROM records
		WHERE name LIKE ? ESCAPE '\'
		AND (expire = 0 OR expire > strftime('%s', CURRENT_TIMESTAMP))
		ORDER BY id
	`, pattern)
	if err != nil {
		return DynamicRecordSet{}, Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) GetRecord(id int) (DynamicRecordSet, error) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	rows, err := sr.db.Query(`
		SELECT id, expire, record FROM records
		WHERE id = ?
		AND (expire = 0 OR expire > strftime('%s', CURRENT_TIMESTAMP))
	`, id)
	if err != nil {
		return DynamicRecordSet{}, Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (sr *SqliteResolver) RemoveRecord(id int) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	result, err := sr.db.Exec(`DELETE FROM records WHERE id = ?`, id)
	if err != nil {
		return Error{TypeInternalError, err, "failed to prepare query"}
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Error{TypeInternalError, err, "failed to get removed record ID"}
	}
	if affected == 0 {
		return ErrNoSuchRecord
	}
	return nil
}

func (sr *SqliteResolver) Resolve(w ResponseWriter, r Request) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	rows, err := sr.db.Query(`
		SELECT record, expire FROM records
		WHERE name = ? AND qtype = ?
		AND (expire = 0 OR expire > strftime('%s', CURRENT_TIMESTAMP))
	`, r.Name, r.QtypeString())
	if err != nil {
		return Error{TypeInternalError, err, "failed to prepare query"}
	}
	defer rows.Close()

	var text string
	var expire int64

	for rows.Next() {
		if err := rows.Scan(&text, &expire); err != nil {
			return Error{TypeExternalError, err, "failed to scan record row"}
		}

		var record Record
		var err error

		if expire != 0 {
			record, err = NewRecordWithExpire(text, time.Unix(expire, 0))
		} else {
			record, err = NewRecord(text)
		}
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
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	close(sr.closer)

	return sr.db.Close()
}
