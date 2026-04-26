"""Pure stats / analysis functions shared between `analyze` and `report`.

Kept as free functions because there is no meaningful state to carry — every
function takes a list of ping dicts (sorted by `_ts`) and returns a value.
No I/O, no logging, no side effects: easy to unit-test, easy to reuse.

Every function assumes:
  * `pings` is a list of dicts with keys:
      ts, target_label, target_kind, target_host, target_port,
      success, rtt_ms, error
  * `_ts` is a parsed `datetime` on each ping (see `parse_pings`).
  * Rows are sorted by `_ts` ascending.
"""

from __future__ import annotations

import contextlib
import json
import statistics
from collections import defaultdict
from collections.abc import Iterable, Iterator
from datetime import datetime
from pathlib import Path

OUTAGE_MIN_CONSECUTIVE = 3


# --------------------------------------------------------------------------
# Loading helpers
# --------------------------------------------------------------------------


def parse_ts(s: str) -> datetime:
    """Accept both `...+00:00` and `...Z` suffixes."""
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


def load_jsonl(path: Path) -> list[dict]:
    records: list[dict] = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            stripped = line.strip()
            if not stripped:
                continue
            with contextlib.suppress(json.JSONDecodeError):
                records.append(json.loads(stripped))
    return records


def parse_pings(pings: list[dict]) -> list[dict]:
    """Attach `_ts` datetime to each ping and sort in place. Returns the same list."""
    for p in pings:
        p["_ts"] = parse_ts(p["ts"])
    pings.sort(key=lambda r: r["_ts"])
    return pings


# --------------------------------------------------------------------------
# Tiny math helpers
# --------------------------------------------------------------------------


def percentile(sorted_xs: list[float], q: float) -> float | None:
    """Linear-index percentile. `sorted_xs` must be sorted ascending."""
    if not sorted_xs:
        return None
    idx = min(int(len(sorted_xs) * q), len(sorted_xs) - 1)
    return sorted_xs[idx]


def jitter_ms(rtts_in_time_order: list[float]) -> float | None:
    """RFC 3550-style jitter: mean absolute RTT delta between consecutive samples.
    This is what voice/video codecs actually feel, unlike std-dev-of-all."""
    if len(rtts_in_time_order) < 2:
        return None

    diffs: list[float] = []
    for i in range(1, len(rtts_in_time_order)):
        delta = rtts_in_time_order[i] - rtts_in_time_order[i - 1]
        diffs.append(abs(delta))
    return round(statistics.mean(diffs), 2)


# --------------------------------------------------------------------------
# Per-target summary
# --------------------------------------------------------------------------


def compute_summary(pings: list[dict]) -> dict:
    """Overall stats + per-target breakdown (uptime %, RTT percentiles, jitter)."""
    if not pings:
        return {
            "window_start_utc": None,
            "window_end_utc": None,
            "duration_hours": 0,
            "total_pings": 0,
            "targets": {},
        }

    by_target = _group_by_target(pings)
    targets = {label: _target_stats(rows) for label, rows in by_target.items()}

    start = pings[0]["_ts"]
    end = pings[-1]["_ts"]
    return {
        "window_start_utc": start.isoformat(),
        "window_end_utc": end.isoformat(),
        "duration_hours": round((end - start).total_seconds() / 3600, 2),
        "total_pings": len(pings),
        "targets": targets,
    }


def _group_by_target(pings: list[dict]) -> dict[str, list[dict]]:
    out: dict[str, list[dict]] = defaultdict(list)
    for p in pings:
        out[p["target_label"]].append(p)
    return out


def _target_stats(rows: list[dict]) -> dict:
    rtts_in_time_order: list[float] = []
    ok = 0
    for r in rows:
        if not r["success"]:
            continue
        ok += 1
        if r["rtt_ms"] is not None:
            rtts_in_time_order.append(r["rtt_ms"])

    rtts_sorted = sorted(rtts_in_time_order)
    total = len(rows)
    head = rows[0]

    uptime_pct = 0.0
    if total:
        uptime_pct = round((ok / total) * 100, 3)

    max_ms: float | None = None
    mean_ms: float | None = None
    if rtts_sorted:
        max_ms = round(max(rtts_sorted), 2)
        mean_ms = round(statistics.mean(rtts_sorted), 2)

    return {
        "kind": head.get("target_kind", "icmp"),
        "host": head.get("target_host", ""),
        "port": head.get("target_port"),
        "total_pings": total,
        "successful": ok,
        "failed": total - ok,
        "uptime_pct": uptime_pct,
        "p50_ms": _round_or_none(percentile(rtts_sorted, 0.5)),
        "p95_ms": _round_or_none(percentile(rtts_sorted, 0.95)),
        "p99_ms": _round_or_none(percentile(rtts_sorted, 0.99)),
        "max_ms": max_ms,
        "mean_ms": mean_ms,
        "jitter_ms": jitter_ms(rtts_in_time_order),
    }


def _round_or_none(v: float | None) -> float | None:
    if v is None:
        return None
    return round(v, 2)


# --------------------------------------------------------------------------
# Outage detection
# --------------------------------------------------------------------------


def find_target_outages(pings: list[dict], label: str) -> list[dict]:
    """Per-target loss bursts (>=3 consecutive failures of one target)."""
    rows = [p for p in pings if p["target_label"] == label]
    return list(_detect_bursts(rows))


def find_full_network_outages(pings: list[dict]) -> list[dict]:
    """Cycles where every target failed simultaneously for >=3 consecutive cycles."""
    cycles = _group_by_cycle(pings)
    synthetic_rows = [_cycle_as_row(ts, cycle) for ts, cycle in cycles]
    return list(_detect_bursts(synthetic_rows))


def _cycle_as_row(ts: datetime, cycle: list[dict]) -> dict:
    """Collapse all probes in one cycle into a single row: success iff at
    least one probe in the cycle succeeded."""
    any_success = False
    for p in cycle:
        if p["success"]:
            any_success = True
            break
    return {
        "_ts": ts,
        "success": any_success,
        "error": _combined_error(cycle),
    }


def _group_by_cycle(pings: list[dict]) -> list[tuple[datetime, list[dict]]]:
    """Group pings by identical `ts`. Each cycle fires all probes with the
    same timestamp, so this recovers cycle boundaries cleanly."""
    buckets: dict[str, list[dict]] = defaultdict(list)
    for p in pings:
        buckets[p["ts"]].append(p)

    result: list[tuple[datetime, list[dict]]] = []
    for ts_str, cycle in buckets.items():
        result.append((parse_ts(ts_str), cycle))
    result.sort(key=lambda pair: pair[0])
    return result


def _combined_error(cycle_pings: list[dict]) -> str:
    errors: set[str] = set()
    for p in cycle_pings:
        if p["success"]:
            continue
        err = p.get("error")
        if err:
            errors.add(err)
    if not errors:
        return "unknown"
    return ", ".join(sorted(errors))


def _detect_bursts(rows: Iterable[dict]) -> Iterator[dict]:
    """Yield a dict per run of `OUTAGE_MIN_CONSECUTIVE`+ consecutive failures.

    Rows must have `_ts`, `success`, and optionally `error`. Expressed as a
    small state machine with early exits so the control flow reads top-down."""
    start: datetime | None = None
    length = 0
    reasons: dict[str, int] = defaultdict(int)
    last_ts: datetime | None = None

    for r in rows:
        ts = r["_ts"]
        last_ts = ts
        if not r["success"]:
            if start is None:
                start = ts
            length += 1
            reasons[r.get("error") or "unknown"] += 1
            continue

        if start is not None and length >= OUTAGE_MIN_CONSECUTIVE:
            yield _burst_dict(start, ts, length, reasons)
        start, length, reasons = None, 0, defaultdict(int)

    if start is not None and length >= OUTAGE_MIN_CONSECUTIVE and last_ts is not None:
        yield _burst_dict(start, last_ts, length, reasons)


def _burst_dict(
    start: datetime, end: datetime, length: int, reasons: dict[str, int]
) -> dict:
    return {
        "start_utc": start.isoformat(),
        "start_local": start.astimezone().strftime("%Y-%m-%d %H:%M:%S"),
        "duration_sec": round((end - start).total_seconds(), 1),
        "consecutive_cycles_failed": length,
        "errors": dict(reasons),
    }


# --------------------------------------------------------------------------
# Hourly buckets (for charts)
# --------------------------------------------------------------------------


def bucket_by_hour(pings: list[dict]) -> list[dict]:
    """Aggregate pings into hourly buckets for readable week-long charts.
    Each bucket carries overall stats plus a per-target sub-breakdown."""
    buckets: dict[str, dict] = defaultdict(_empty_bucket)
    for p in pings:
        hour_key = p["_ts"].astimezone().strftime("%Y-%m-%d %H:00")
        _accumulate_into_bucket(buckets[hour_key], p)
    return [_finalize_bucket(k, b) for k, b in sorted(buckets.items())]


def _empty_bucket() -> dict:
    return {
        "total": 0,
        "fail": 0,
        "rtts": [],
        "by_target": defaultdict(lambda: {"total": 0, "fail": 0}),
    }


def _accumulate_into_bucket(b: dict, p: dict) -> None:
    b["total"] += 1
    target = b["by_target"][p["target_label"]]
    target["total"] += 1
    if not p["success"]:
        b["fail"] += 1
        target["fail"] += 1
        return
    if p["rtt_ms"] is not None:
        b["rtts"].append(p["rtt_ms"])


def _finalize_bucket(hour: str, b: dict) -> dict:
    total = b["total"]
    fail = b["fail"]
    rtts = sorted(b["rtts"])

    fail_pct = 0.0
    uptime_pct = 0.0
    if total:
        fail_pct = round((fail / total) * 100, 2)
        uptime_pct = round(((total - fail) / total) * 100, 2)

    median_rtt: float | None = None
    if rtts:
        median_rtt = round(statistics.median(rtts), 1)

    # p95 only meaningful once we have a handful of samples.
    p95_rtt: float | None = None
    if len(rtts) > 20:
        p95_rtt = round(rtts[int(len(rtts) * 0.95)], 1)

    target_breakdown: dict[str, dict] = {}
    for label, target_bucket in b["by_target"].items():
        target_breakdown[label] = _finalize_target_bucket(target_bucket)

    return {
        "hour": hour,
        "total": total,
        "fail": fail,
        "fail_pct": fail_pct,
        "uptime_pct": uptime_pct,
        "median_rtt": median_rtt,
        "p95_rtt": p95_rtt,
        "by_target": target_breakdown,
    }


def _finalize_target_bucket(t: dict) -> dict:
    total = t["total"]
    fail = t["fail"]
    uptime_pct = 0.0
    if total:
        uptime_pct = round(((total - fail) / total) * 100, 2)
    return {
        "total": total,
        "fail": fail,
        "uptime_pct": uptime_pct,
    }
