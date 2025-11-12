package db

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

type Config struct {
	DBMS     string
	DSN      string
	TraceSQL bool
}

func Open(cfg *Config) (*bun.DB, error) {
	var ret *bun.DB

	switch cfg.DBMS {
	case "sqlite", "sqlite3":
		sqldb, err := sql.Open(sqliteshim.ShimName, cfg.DSN)
		if err != nil {
			return nil, err
		}

		ret = bun.NewDB(sqldb, sqlitedialect.New())
	default:
		return nil, fmt.Errorf("unsupported DBMS: %s", cfg.DBMS)
	}

	if cfg.TraceSQL {
		ret.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
		))
	}

	return ret, nil
}
