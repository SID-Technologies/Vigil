"""Unit tests for the stats pipeline.

All pure-function tests — no fixtures on disk, no network. Each test uses the
smallest synthetic input that can possibly fail the behavior under test.
"""

from __future__ import annotations

from pingscraper.stats import (
    compute_summary,
    find_full_network_outages,
    find_target_outages,
    jitter_ms,
    percentile,
)

# --------------------------------------------------------------------------
# Jitter
# --------------------------------------------------------------------------


def test_jitter_empty_returns_none():
    assert jitter_ms([]) is None


def test_jitter_single_sample_returns_none():
    assert jitter_ms([42.0]) is None


def test_jitter_known_series():
    # Diffs: |12-10|=2, |11-12|=1, |14-11|=3  ->  mean = 2.0
    assert jitter_ms([10.0, 12.0, 11.0, 14.0]) == 2.0


def test_jitter_constant_series_is_zero():
    assert jitter_ms([5.0, 5.0, 5.0, 5.0]) == 0.0


# --------------------------------------------------------------------------
# Percentile
# --------------------------------------------------------------------------


def test_percentile_empty_returns_none():
    assert percentile([], 0.5) is None


def test_percentile_midpoint():
    # 10 samples, 0.5 -> index 5 -> value 6.0 (1-indexed 6th)
    xs = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
    assert percentile(xs, 0.5) == 6.0


def test_percentile_top_clamps_to_last():
    # 1.0 would be out of range; must clamp to last element.
    assert percentile([1.0, 2.0, 3.0], 1.0) == 3.0


# --------------------------------------------------------------------------
# Per-target outages (≥3 consecutive failures for one target)
# --------------------------------------------------------------------------


def test_find_target_outages_detects_single_burst(mixed_success_pings):
    bursts = find_target_outages(mixed_success_pings, "t1")
    assert len(bursts) == 1
    assert bursts[0]["consecutive_cycles_failed"] == 4
    assert bursts[0]["errors"] == {"timeout": 4}


def test_find_target_outages_ignores_short_runs(start_ts):
    # Only 2 consecutive failures — below OUTAGE_MIN_CONSECUTIVE = 3.
    from datetime import timedelta

    from tests.conftest import _mk_ping

    rows = [
        _mk_ping(start_ts, "t1", True, 10.0),
        _mk_ping(start_ts + timedelta(seconds=1), "t1", False, None),
        _mk_ping(start_ts + timedelta(seconds=2), "t1", False, None),
        _mk_ping(start_ts + timedelta(seconds=3), "t1", True, 10.0),
    ]
    from pingscraper.stats import parse_pings

    parse_pings(rows)
    assert find_target_outages(rows, "t1") == []


def test_find_target_outages_trailing_burst_still_counted(start_ts):
    """A failure burst at the very end of the log (no recovery row) must still
    be reported, not silently dropped."""
    from datetime import timedelta

    from pingscraper.stats import parse_pings
    from tests.conftest import _mk_ping

    rows = [_mk_ping(start_ts, "t1", True, 10.0)]
    for i in range(1, 5):
        rows.append(
            _mk_ping(start_ts + timedelta(seconds=i), "t1", False, None, error="timeout")
        )
    parse_pings(rows)

    bursts = find_target_outages(rows, "t1")
    assert len(bursts) == 1
    assert bursts[0]["consecutive_cycles_failed"] == 4


# --------------------------------------------------------------------------
# Full-network outages (all targets fail simultaneously)
# --------------------------------------------------------------------------


def test_find_full_network_outages_detects_joint_failure(two_target_pings):
    outages = find_full_network_outages(two_target_pings)
    assert len(outages) == 1
    assert outages[0]["consecutive_cycles_failed"] == 3


def test_find_full_network_outages_skips_when_any_target_up(start_ts):
    """If one target keeps succeeding during the other's outage, it is not a
    full-network outage."""
    from datetime import timedelta

    from pingscraper.stats import parse_pings
    from tests.conftest import _mk_ping

    rows: list[dict] = []
    for i in range(6):
        ts = start_ts + timedelta(seconds=i * 2)
        a_success = True  # 'a' stays up
        b_success = i < 2 or i > 4  # 'b' fails cycles 2..4
        rows.append(_mk_ping(ts, "a", a_success, 10.0 if a_success else None))
        rows.append(_mk_ping(ts, "b", b_success, 20.0 if b_success else None))
    parse_pings(rows)

    assert find_full_network_outages(rows) == []


# --------------------------------------------------------------------------
# Summary shape
# --------------------------------------------------------------------------


def test_compute_summary_empty_pings():
    summary = compute_summary([])
    assert summary["total_pings"] == 0
    assert summary["targets"] == {}


def test_compute_summary_single_target(mixed_success_pings):
    summary = compute_summary(mixed_success_pings)
    assert summary["total_pings"] == len(mixed_success_pings)
    assert set(summary["targets"].keys()) == {"t1"}

    t = summary["targets"]["t1"]
    assert t["total_pings"] == 17
    assert t["successful"] == 13
    assert t["failed"] == 4
    assert 0 < t["uptime_pct"] < 100
    assert t["p50_ms"] is not None
    assert t["jitter_ms"] is not None
