// Package db wires the Ent client to SQLite under the user's app data dir.
// Uses modernc.org/sqlite (pure Go) so cross-compilation needs no C toolchain.
package db

import (
	"context"
	"database/sql"
	"path/filepath"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/sid-technologies/vigil/db/ent"

	_ "modernc.org/sqlite" // registers "sqlite" driver
)

// Open opens or creates <dataDir>/vigil.db and runs schema migrations.
// WAL mode for concurrent reads during writes; FKs on for Ent cascades;
// busy_timeout=5000 absorbs brief write contention.
func Open(ctx context.Context, dataDir string) (*ent.Client, error) {
	dbPath := filepath.Join(dataDir, "vigil.db")
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}
	// SQLite serializes writes anyway; pinning to one conn dodges Ent opening
	// 10 conns and timing out the 11th under load.
	rawDB.SetMaxOpenConns(1)

	drv := entsql.OpenDB(dialect.SQLite, rawDB)
	client := ent.NewClient(ent.Driver(drv))

	err = client.Schema.Create(ctx)
	if err != nil {
		_ = client.Close()
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return client, nil
}
