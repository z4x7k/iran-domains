package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/z4x7k/iran-domains-tg-bot/db/gen/model"
	"github.com/z4x7k/iran-domains-tg-bot/db/gen/table"
)

func ExecPragmas(ctx context.Context, db *sql.DB) error {
	pragmas := [][]string{
		{"journal_mode", "wal"},
		{"wal_autocheckpoint", "0"},
		{"synchronous", "3"},
		{"locking_mode", "exclusive"},
		{"journal_size_limit", "-1"},
		{"checkpoint_fullfsync", "1"},
		{"fullfsync", "1"},
	}
	for _, v := range pragmas {
		k, v := v[0], v[1]
		pragma := fmt.Sprintf("PRAGMA %s = %s;", k, v)
		if _, err := db.ExecContext(ctx, pragma); nil != err {
			return fmt.Errorf("unable to execute pragma '%s': %v", pragma, err)
		}

		pragma = fmt.Sprintf("PRAGMA %s;", k)
		row := db.QueryRowContext(ctx, pragma)
		if row == nil {
			return fmt.Errorf("unexpected nil row returned from pragma check query: %s", pragma)
		}
		var val string
		if err := row.Scan(&val); nil != err {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("expected pragma value to be returned for %s, got no rows", k)
			}
			return fmt.Errorf("unable to query pragma value for %s: %v", k, err)
		}
		if val != v {
			return fmt.Errorf("pragma '%s' updated value is not equal to previously set expected value. expected %s, got: %v", k, v, val)
		}
	}

	return nil
}

var (
	ErrDuplicateDomain = errors.New("domain already exists")
	ErrBusy            = errors.New("database is busy at the moment. try again later")
)

func InsertDomain(ctx context.Context, db *sql.DB, domain string, userID int64) error {
	now := time.Now().UTC().Unix()
	res, err := table.Domains.
		INSERT(table.Domains.AllColumns).
		MODEL(model.Domains{Domain: domain, CreatedTs: now, CreatedByID: userID}).
		ExecContext(ctx, db)
	if nil != err {
		var sqlErr sqlite3.Error
		if errors.As(err, &sqlErr) {

			if sqlErr.Code == sqlite3.ErrConstraint && sqlErr.Error() == "UNIQUE constraint failed: domains.domain" {
				return ErrDuplicateDomain
			}
		}
		return fmt.Errorf("db: failed to insert domain into database: %v", err)
	} else if affectedRows, err := res.RowsAffected(); nil != err {
		return fmt.Errorf("db: failed to get number of affected rows by domain insert query: %v", err)
	} else if affectedRows != 1 {
		return fmt.Errorf("expected 1 row to be affected by domain insert query, got %d", affectedRows)
	}

	return nil
}
