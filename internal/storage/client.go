// Package storage is the persistence layer above Ent.
package storage

import "github.com/sid-technologies/vigil/db/ent"

// Client is the storage-layer entry point. Composes per-resource sub-
// clients, each of which encapsulates its own Ent table interactions.
type Client struct {
	Samples *SampleClient
	Outages *OutageClient
	Targets *TargetClient
	Config  *ConfigClient
	Wifi    *WifiClient
	Seed    *SeedClient
}

// NewClient wraps an Ent client.
func NewClient(entClient *ent.Client) *Client {
	return &Client{
		Samples: NewSampleClient(entClient),
		Outages: NewOutageClient(entClient),
		Targets: NewTargetClient(entClient),
		Config:  NewConfigClient(entClient),
		Wifi:    NewWifiClient(entClient),
		Seed:    NewSeedClient(entClient),
	}
}
