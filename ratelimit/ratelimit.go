package ratelimit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/mattn/go-sqlite3"

	"github.com/z4x7k/iran-domains-tg-bot/db"
	"github.com/z4x7k/iran-domains-tg-bot/db/gen/model"
	"github.com/z4x7k/iran-domains-tg-bot/db/gen/table"
)

type RateLimiter struct {
	db          *sql.DB
	maxAttempts int
	interval    time.Duration
}

func New(db *sql.DB, MaxAttempts int, Interval time.Duration) RateLimiter {
	return RateLimiter{
		db:          db,
		maxAttempts: MaxAttempts,
		interval:    Interval,
	}
}

func (r *RateLimiter) CanPass(ctx context.Context, userID int64) (bool, error) {
	now := time.Now().UTC().Unix()
	intervalMillis := int64(r.interval.Seconds())
	query, args := table.UsersRateLimit.
		INSERT(table.UsersRateLimit.AllColumns).
		MODEL(model.UsersRateLimit{TheUserID: userID, LastAccessTs: now, TheCount: 0}).
		ON_CONFLICT(table.UsersRateLimit.TheUserID).
		DO_UPDATE(
			sqlite.
				SET(
					table.UsersRateLimit.LastAccessTs.SET(
						sqlite.
							CAST(
								sqlite.
									CASE().
									WHEN(
										sqlite.OR(
											sqlite.Int64(now-intervalMillis).GT(table.UsersRateLimit.LastAccessTs),
											sqlite.AND(
												sqlite.Int64(now-intervalMillis).LT(table.UsersRateLimit.LastAccessTs),
												table.UsersRateLimit.TheCount.LT(sqlite.Int(int64(r.maxAttempts))),
											),
										),
									).
									THEN(sqlite.Int64(now)).
									ELSE(table.UsersRateLimit.LastAccessTs),
							).
							AS_INTEGER(),
					),
					table.UsersRateLimit.TheCount.SET(
						sqlite.
							CAST(
								sqlite.
									CASE().
									WHEN(
										sqlite.Int64(now-intervalMillis).GT(table.UsersRateLimit.LastAccessTs),
									).
									THEN(sqlite.Int64(0)).
									WHEN(
										sqlite.AND(
											sqlite.Int64(now-intervalMillis).LT(table.UsersRateLimit.LastAccessTs),
											table.UsersRateLimit.TheCount.LT(sqlite.Int(int64(r.maxAttempts))),
										),
									).
									THEN(table.UsersRateLimit.TheCount.ADD(sqlite.Int(1))).
									ELSE(table.UsersRateLimit.TheCount),
							).
							AS_INTEGER(),
					),
				),
		).
		RETURNING(table.UsersRateLimit.TheCount).
		Sql()
	var theCount int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&theCount); nil != err {
		var sqlErr sqlite3.Error
		if errors.As(err, &sqlErr) {
			if sqlErr.Code == sqlite3.ErrBusy && sqlErr.Error() == "database is locked" {
				return false, db.ErrBusy
			}
		}
		return false, fmt.Errorf("db: failed to query user rate limit counter: %v", err)
	}
	fmt.Println(theCount) // TODO: remove me!
	return theCount < r.maxAttempts, nil
}
