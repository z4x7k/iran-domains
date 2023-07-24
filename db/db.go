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
