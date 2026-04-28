package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
)

// AppConfig is the JSON-serialized view of the singleton app_config row.
type AppConfig struct {
	PingIntervalSec   float64 `json:"ping_interval_sec"`
	FlushIntervalSec  int     `json:"flush_interval_sec"`
	PingTimeoutMs     int     `json:"ping_timeout_ms"`
	RetentionRawDays  int     `json:"retention_raw_days"`
	Retention1minDays int     `json:"retention_1min_days"`
	Retention5minDays int     `json:"retention_5min_days"`
	WifiSampleEnabled bool    `json:"wifi_sample_enabled"`
}

// AppConfigSingletonID is the row id of the singleton app_config row.
// The table is enforced by application code to hold exactly one row.
const AppConfigSingletonID = 1

// GetAppConfig returns the singleton config row. Caller must have called
// SeedAppConfig once at startup.
func GetAppConfig(ctx context.Context, client *ent.Client) (AppConfig, error) {
	row, err := client.AppConfig.Get(ctx, AppConfigSingletonID)
	if err != nil {
		return AppConfig{}, err //nolint:wrapcheck // wrapped at IPC boundary
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

// UpdateAppConfig applies a partial patch. Each pointer is optional — nil
// means "leave unchanged". Returns the new full config.
func UpdateAppConfig(ctx context.Context, client *ent.Client, patch AppConfigPatch) (AppConfig, error) {
	upd := client.AppConfig.UpdateOneID(AppConfigSingletonID)
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
		return AppConfig{}, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return GetAppConfig(ctx, client)
}

// AppConfigPatch is the partial-update payload — pointers so unset fields
// can be distinguished from intentional zeros.
type AppConfigPatch struct {
	PingIntervalSec   *float64 `json:"ping_interval_sec,omitempty"`
	FlushIntervalSec  *int     `json:"flush_interval_sec,omitempty"`
	PingTimeoutMs     *int     `json:"ping_timeout_ms,omitempty"`
	RetentionRawDays  *int     `json:"retention_raw_days,omitempty"`
	Retention1minDays *int     `json:"retention_1min_days,omitempty"`
	Retention5minDays *int     `json:"retention_5min_days,omitempty"`
	WifiSampleEnabled *bool    `json:"wifi_sample_enabled,omitempty"`
}
