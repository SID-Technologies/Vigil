package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/pkg/errors"
)

// AppConfig is the JSON view of the singleton app_config row.
type AppConfig struct {
	PingIntervalSec   float64 `json:"ping_interval_sec"`
	FlushIntervalSec  int     `json:"flush_interval_sec"`
	PingTimeoutMs     int     `json:"ping_timeout_ms"`
	RetentionRawDays  int     `json:"retention_raw_days"`
	Retention1minDays int     `json:"retention_1min_days"`
	Retention5minDays int     `json:"retention_5min_days"`
	WifiSampleEnabled bool    `json:"wifi_sample_enabled"`
}

// AppConfigSingletonID — app_config holds exactly one row, enforced in code.
const AppConfigSingletonID = 1

// AppConfigPatch — pointer fields distinguish "unset" from intentional zero.
type AppConfigPatch struct {
	PingIntervalSec   *float64 `json:"ping_interval_sec,omitempty"`
	FlushIntervalSec  *int     `json:"flush_interval_sec,omitempty"`
	PingTimeoutMs     *int     `json:"ping_timeout_ms,omitempty"`
	RetentionRawDays  *int     `json:"retention_raw_days,omitempty"`
	Retention1minDays *int     `json:"retention_1min_days,omitempty"`
	Retention5minDays *int     `json:"retention_5min_days,omitempty"`
	WifiSampleEnabled *bool    `json:"wifi_sample_enabled,omitempty"`
}

// ConfigClient owns the singleton app_config row.
type ConfigClient struct {
	client *ent.Client
}

// NewConfigClient wraps an Ent client.
func NewConfigClient(client *ent.Client) *ConfigClient {
	return &ConfigClient{client: client}
}

// Get returns the singleton app_config row.
func (c *ConfigClient) Get(ctx context.Context) (AppConfig, error) {
	row, err := c.client.AppConfig.Get(ctx, AppConfigSingletonID)
	if err != nil {
		return AppConfig{}, errors.Wrap(err, "failed to get app config")
	}

	return AppConfig{
		PingIntervalSec:   row.PingIntervalSec,
		FlushIntervalSec:  row.FlushIntervalSec,
		PingTimeoutMs:     row.PingTimeoutMs,
		RetentionRawDays:  row.RetentionRawDays,
		Retention1minDays: row.Retention1minDays,
		Retention5minDays: row.Retention5minDays,
		WifiSampleEnabled: row.WifiSampleEnabled,
	}, nil
}

// Update applies a partial patch (nil fields untouched) and returns the new full config.
func (c *ConfigClient) Update(ctx context.Context, patch AppConfigPatch) (AppConfig, error) {
	upd := c.client.AppConfig.UpdateOneID(AppConfigSingletonID)
	if patch.PingIntervalSec != nil {
		upd.SetPingIntervalSec(*patch.PingIntervalSec)
	}

	if patch.FlushIntervalSec != nil {
		upd.SetFlushIntervalSec(*patch.FlushIntervalSec)
	}

	if patch.PingTimeoutMs != nil {
		upd.SetPingTimeoutMs(*patch.PingTimeoutMs)
	}

	if patch.RetentionRawDays != nil {
		upd.SetRetentionRawDays(*patch.RetentionRawDays)
	}

	if patch.Retention1minDays != nil {
		upd.SetRetention1minDays(*patch.Retention1minDays)
	}

	if patch.Retention5minDays != nil {
		upd.SetRetention5minDays(*patch.Retention5minDays)
	}

	if patch.WifiSampleEnabled != nil {
		upd.SetWifiSampleEnabled(*patch.WifiSampleEnabled)
	}

	_, err := upd.Save(ctx)
	if err != nil {
		return AppConfig{}, errors.Wrap(err, "failed to update app config")
	}

	return c.Get(ctx)
}
