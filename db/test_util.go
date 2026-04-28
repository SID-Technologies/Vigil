package db

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/sid-technologies/vigil/db/ent"

	_ "modernc.org/sqlite" // registers "sqlite" driver
)

var testDBCounter atomic.Int64

// migrateMu — Atlas's setupTables/CopyTables share global state and race when
// multiple t.Parallel() tests call Schema.Create concurrently.
var migrateMu sync.Mutex

const decimalBase = 10

// SetupTestEntClientDB returns an isolated in-memory Ent client per test.
func SetupTestEntClientDB(t *testing.T) *ent.Client {
	t.Helper()

	id := testDBCounter.Add(1)
	dsn := "file:vigil_test_" + intToString(id) + "?mode=memory&cache=shared&_pragma=foreign_keys(1)"

	migrateMu.Lock()
	defer migrateMu.Unlock()

	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open test sqlite: %v", err)
	}

	rawDB.SetMaxOpenConns(1)

	drv := entsql.OpenDB(dialect.SQLite, rawDB)
	client := ent.NewClient(ent.Driver(drv))

	err = client.Schema.Create(context.Background())
	if err != nil {
		_ = client.Close()

		t.Fatalf("schema create: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte

	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%decimalBase)
		n /= decimalBase
	}

	return string(buf[pos:])
}
