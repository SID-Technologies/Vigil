package monitor

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	entsample "github.com/sid-technologies/vigil/db/ent/sample"
	"github.com/sid-technologies/vigil/db/ent/wifisample"
	"github.com/sid-technologies/vigil/internal/constants"
	"github.com/sid-technologies/vigil/internal/netinfo"
	"github.com/sid-technologies/vigil/internal/probes"
)

// configProvider is the slice of Monitor's API the flusher needs. Decoupled
// so flusher tests can mock it without spinning up a full Monitor.
type configProvider interface {
	Config() Config
	subscribe() <-chan struct{}
}

// flusher drains the in-memory buffer to SQLite on a fixed interval and
// captures one Wi-Fi sample per flush. Bulk inserts use Ent's CreateBulk
// for ~10× throughput vs individual saves.
//
// On insert failure (rare with SQLite — usually disk full or DB locked too
// long), results are requeued and we log a warning. Next flush will retry.
//
// The flush interval and wifi-enabled flag are read fresh from the
// configProvider each iteration, so config changes via UpdateConfig take
// effect on the very next flush. The wake channel from subscribe()
// shortens that further by interrupting in-progress sleeps.
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

// run blocks until ctx is canceled. Re-times the loop each iteration so
// flush_interval changes propagate without restart.
func (f *flusher) run(ctx context.Context) {
	wake := f.cfg.subscribe()

	for {
		interval := time.Duration(f.cfg.Config().FlushIntervalSec) * time.Second
		if interval <= 0 {
			// Safety guard: if the user somehow wrote a non-positive flush
			// interval, fall back to the seed default so the flusher keeps
			// running instead of busy-looping on a zero timer.
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
			// Config changed — abandon the current sleep, loop again to
			// pick up the new interval. We do NOT flush here; the wake is
			// purely about reschedule, not "flush now."
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

// Compile-time anchor — keeps unused-import warnings honest.
var (
	_ = entsample.FieldTsUnixMs
	_ = wifisample.FieldTsUnixMs
)
