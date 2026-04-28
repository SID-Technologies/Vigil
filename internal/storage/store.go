package storage

import "github.com/sid-technologies/vigil/db/ent"

// Store is the storage-layer entry point. It owns the Ent client and
// exposes typed query/mutation methods so callers don't thread the raw
// client through every layer.
type Store struct {
	client *ent.Client
}

// NewStore wraps an Ent client.
func NewStore(client *ent.Client) *Store {
	return &Store{client: client}
}

// Client returns the underlying Ent client. Use sparingly — prefer the
// typed methods on Store. Exposed for the rare caller that needs raw
// access (e.g. test setup, schema migration in app startup).
func (s *Store) Client() *ent.Client {
	return s.client
}
