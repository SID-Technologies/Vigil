package db

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/sid-technologies/vigil/db/ent"

	_ "modernc.org/sqlite"
)

var testDBCounter atomic.Int64

// decimalBase is the radix used by intToString below. Named so the call
// site reads as base-10 conversion rather than a stray literal.
const decimalBase = 10

// SetupTestEntClientDB returns an isolated in-memory SQLite Ent client for
// the duration of the test. Each test gets its own database via a unique
// `cache=shared` URI so parallel tests don't clobber each other. Schema
// migrations run automatically; cleanup is registered with t.Cleanup.
func SetupTestEntClientDB(t *testing.T) *ent.Client {
	t.Helper()

	id := testDBCounter.Add(1)
	dsn := "file:vigil_test_" + intToString(id) + "?mode=memory&cache=shared&_pragma=foreign_keys(1)"

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
