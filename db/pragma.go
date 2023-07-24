package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
