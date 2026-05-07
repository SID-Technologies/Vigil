package monitor

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/constants"
	"github.com/sid-technologies/vigil/internal/netinfo"
	"github.com/sid-technologies/vigil/internal/probes"
)

// configProvider is the slice of Monitor's API the flusher needs — kept
// narrow so tests can mock without a full Monitor.
type configProvider interface {
	Config() Config
	subscribe() <-chan struct{}
}

// flusher drains the buffer to SQLite on FlushIntervalSec and captures one
// Wi-Fi sample per flush. Bulk-insert failures requeue the batch.
type flusher struct {
	client    *ent.Client
	buf       *buffer
	cfg       configProvider
	onResults func([]probes.Result) // optional, used for IPC events
}

func newFlusher(client *ent.Client, buf *buffer, cfg configProvider) *flusher {
	return &flusher{
		client: client,
		buf:    buf,
		cfg:    cfg,
	}
}

func (f *flusher) run(ctx context.Context) {
	wake := f.cfg.subscribe()

	for {
		interval := time.Duration(f.cfg.Config().FlushIntervalSec) * time.Second
		if interval <= 0 {
			// Guard against a non-positive value busy-looping the timer.
			interval = time.Duration(constants.DefaultFlushIntervalSec) * time.Second
		}

		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			// Final flush so we don't lose the last partial cycle.
			f.flushOnce(context.Background()) //nolint:contextcheck // final flush must complete despite parent ctx being canceled

			return
		case <-timer.C:
			f.flushOnce(ctx)
		case <-wake:
			// Reschedule only — wake is not "flush now."
			timer.Stop()
		}
	}
}

func (f *flusher) flushOnce(ctx context.Context) {
	if f.cfg.Config().WifiSampleEnabled {
		f.flushWifi(ctx)
	}

	f.flushSamples(ctx)
}

func (f *flusher) flushWifi(ctx context.Context) {
	sample := netinfo.SampleWifi(ctx)

	create := f.client.WifiSample.Create().
		SetTsUnixMs(sample.Timestamp.UnixMilli())
	if sample.SSID != nil {
		create.SetSsid(*sample.SSID)
	}

	if sample.BSSID != nil {
		create.SetBssid(*sample.BSSID)
	}

	if sample.SignalPercent != nil {
		create.SetSignalPercent(*sample.SignalPercent)
	}

	if sample.RSSIDbm != nil {
		create.SetRssiDbm(*sample.RSSIDbm)
	}

	if sample.RxRateMbps != nil {
		create.SetRxRateMbps(*sample.RxRateMbps)
	}

	if sample.TxRateMbps != nil {
		create.SetTxRateMbps(*sample.TxRateMbps)
	}

	if sample.Channel != nil {
		create.SetChannel(*sample.Channel)
	}

	_, err := create.Save(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("flusher: wifi sample save failed")
	}
}

func (f *flusher) flushSamples(ctx context.Context) {
	results := f.buf.drain()
	if len(results) == 0 {
		return
	}

	bulk := make([]*ent.SampleCreate, 0, len(results))
	for _, r := range results {
		c := f.client.Sample.Create().
			SetTsUnixMs(r.TimestampMs).
			SetTargetLabel(r.Target.Label).
			SetTargetKind(string(r.Target.Kind)).
			SetTargetHost(r.Target.Host).
			SetSuccess(r.Success)
		if r.Target.Port != nil {
			c.SetTargetPort(*r.Target.Port)
		}

		if r.RTTMs != nil {
			c.SetRttMs(*r.RTTMs)
		}

		if r.Error != nil {
			c.SetError(*r.Error)
		}

		bulk = append(bulk, c)
	}

	_, err := f.client.Sample.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		log.Error().Err(err).Int("count", len(results)).Msg("flusher: bulk insert failed, requeuing")
		f.buf.requeue(results)

		return
	}

	ok := 0

	for _, r := range results {
		if r.Success {
			ok++
		}
	}

	log.Info().
		Int("flushed", len(results)).
		Int("ok", ok).
		Int("fail", len(results)-ok).
		Int("buffer_remaining", f.buf.len()).
		Msg("flushed probe samples")

	if f.onResults != nil {
		f.onResults(results)
	}
}
