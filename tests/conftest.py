"""Shared pytest fixtures.

Synthetic data builders below so individual tests don't re-invent them.
Each builder returns a fresh list so tests can mutate without side-effects.
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import pytest

from pingscraper.stats import parse_pings


def _mk_ping(
    ts: datetime,
    label: str,
    success: bool,
    rtt_ms: float | None,
    kind: str = "icmp",
    host: str = "1.2.3.4",
    port: int | None = None,
    error: str | None = None,
) -> dict:
    return {
        "ts": ts.isoformat(timespec="milliseconds"),
        "target_label": label,
        "target_kind": kind,
        "target_host": host,
        "target_port": port,
        "success": success,
        "rtt_ms": rtt_ms,
        "error": error,
    }


@pytest.fixture
def start_ts() -> datetime:
    return datetime(2026, 4, 18, 12, 0, 0, tzinfo=timezone.utc)


@pytest.fixture
def mixed_success_pings(start_ts: datetime) -> list[dict]:
    """One target. 10 successes, then 4 failures (one burst), then 3 successes."""
    rows: list[dict] = []
    for i in range(10):
        rows.append(
            _mk_ping(start_ts + timedelta(seconds=i), "t1", True, 10.0 + i)
        )
    for i in range(10, 14):
        rows.append(
            _mk_ping(
                start_ts + timedelta(seconds=i), "t1", False, None, error="timeout"
            )
        )
    for i in range(14, 17):
        rows.append(
            _mk_ping(start_ts + timedelta(seconds=i), "t1", True, 11.0)
        )
    parse_pings(rows)
    return rows


@pytest.fixture
def two_target_pings(start_ts: datetime) -> list[dict]:
    """Two targets sampled at the same timestamps, so `find_full_network_outages`
    has something to chew on."""
    rows: list[dict] = []
    for i in range(5):
        ts = start_ts + timedelta(seconds=i * 2)
        rows.append(_mk_ping(ts, "a", True, 10.0))
        rows.append(_mk_ping(ts, "b", True, 20.0))
    # 3 cycles where both fail together (full-network outage)
    for i in range(5, 8):
        ts = start_ts + timedelta(seconds=i * 2)
        rows.append(_mk_ping(ts, "a", False, None, error="timeout"))
        rows.append(_mk_ping(ts, "b", False, None, error="timeout"))
    # Recovery
    for i in range(8, 10):
        ts = start_ts + timedelta(seconds=i * 2)
        rows.append(_mk_ping(ts, "a", True, 10.0))
        rows.append(_mk_ping(ts, "b", True, 20.0))
    parse_pings(rows)
    return rows
