// Package db wires the Ent client to a SQLite database under the user's app
// data directory.
//
// We use modernc.org/sqlite (pure Go) instead of mattn/go-sqlite3 (cgo) so
// cross-compilation is trivial — `GOOS=windows go build` Just Works on a Mac
// without setting up a Windows C toolchain. Modest perf hit vs cgo, but
// Vigil writes ~5 rows/sec, well below where it matters.
package db

import (
	"context"
	"database/sql"
	"path/filepath"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/sid-technologies/vigil/db/ent"

	// Pure-Go SQLite driver registered as "sqlite".
	_ "modernc.org/sqlite"
)

// Open opens (or creates) the Vigil SQLite database under dataDir and runs
// schema migrations. The DB lives at <dataDir>/vigil.db.
//
// Connection settings:
//   - WAL mode: better concurrent reads while the monitor flushes writes.
//   - foreign_keys=on: Ent relies on FK enforcement for cascades.
//   - busy_timeout=5000: handle brief contention without erroring out.
func Open(ctx context.Context, dataDir string) (*ent.Client, error) {
	dbPath := filepath.Join(dataDir, "vigil.db")
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	// SQLite is single-writer; multiple writers serialize anyway. One
	// connection avoids contention surprises and is plenty for a desktop tool.
	rawDB.SetMaxOpenConns(1)

	drv := entsql.OpenDB(dialect.SQLite, rawDB)
	client := ent.NewClient(ent.Driver(drv))

	if err := client.Schema.Create(ctx); err != nil {
		_ = client.Close()
		return nil, err //nolint:wrapcheck
	}
	return client, nil
}
