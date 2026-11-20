package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	_ "modernc.org/sqlite"
)

// postgresMaxConns is the size of a replica's connection pool to PostgreSQL.
const postgresMaxConns = 25

type Store struct {
	db *bun.DB
}

// Close closes the connections to the database. An in-memory SQLite database is gone
// with its last connection.
func (store *Store) Close() error {
	return store.db.Close()
}

// NewPostgres connects to PostgreSQL, e.g. with
// dsn "postgres://postgres:mysecretpassword@localhost:5432/darts-o-mat?sslmode=disable".
//
// docker run --name some-postgres -e POSTGRES_PASSWORD=mysecretpassword -e POSTGRES_DB=darts-o-mat -p 5432:5432 postgres
func NewPostgres(ctx context.Context, dsn string) (*Store, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	// Without a limit, several replicas under load use up PostgreSQL's
	// max_connections (100 by default); requests beyond the limit wait for a
	// free connection instead. Keeping as many idle as open avoids connecting
	// anew after every burst (database/sql keeps only 2 by default).
	sqldb.SetMaxOpenConns(postgresMaxConns)
	sqldb.SetMaxIdleConns(postgresMaxConns)
	return newStore(ctx, bun.NewDB(sqldb, pgdialect.New()))
}

// NewSQLite opens the SQLite database at dsn, e.g. "file:darts-o-mat.db".
//
// It turns on the write-ahead log with synchronous=NORMAL: with the rollback journal
// every commit waits for several fsyncs (about 90 ms per dart on a disk, held under the
// lock of every change), with these about 0.2 ms. A crash of the computer (not of the
// program) may lose the last commits then, but the database stays intact. Pragmas given
// in dsn come after these, so they win.
func NewSQLite(ctx context.Context, dsn string) (*Store, error) {
	path, query, _ := strings.Cut(dsn, "?")
	pragmas := "_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	if query != "" {
		pragmas += "&" + query
	}
	sqldb, err := sql.Open("sqlite", path+"?"+pragmas)
	if err != nil {
		return nil, err
	}
	// SQLite allows only one writer at a time.
	sqldb.SetMaxOpenConns(1)
	return newStore(ctx, bun.NewDB(sqldb, sqlitedialect.New()))
}

func newStore(ctx context.Context, db *bun.DB) (*Store, error) {
	if err := migrateSchema(ctx, db); err != nil {
		return nil, err
	}
	return &Store{
		db: db,
	}, nil
}
