package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/schema"
)

type Config struct {
	DBMS     string
	DSN      string
	TraceSQL bool
}

func Open(cfg *Config) (*bun.DB, error) {
	var ret *bun.DB
	var sqldb *sql.DB
	var err error
	var dialect schema.Dialect

	switch cfg.DBMS {
	case "mysql", "mariadb":
		sqldb, err = sql.Open("mysql", cfg.DSN)
		dialect = mysqldialect.New()
	case "sqlite", "sqlite3":
		sqldb, err = sql.Open(sqliteshim.ShimName, cfg.DSN)
		dialect = sqlitedialect.New()
	case "postgres", "pq", "pgx", "pg":
		sqldb, err = sql.Open("pgx", cfg.DSN)
		dialect = pgdialect.New()
	default:
		return nil, fmt.Errorf("unsupported DBMS: %s", cfg.DBMS)
	}

	if err != nil {
		return nil, err
	}

	ret = bun.NewDB(sqldb, dialect)

	if cfg.TraceSQL {
		ret.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
		))
	}

	return ret, nil
}

// ExecTx executes provided SQL statements inside a single transaction using
// the provided Context and DB connection.
func ExecTx(ctx context.Context, db *bun.DB, opts *sql.TxOptions, statements []string) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
