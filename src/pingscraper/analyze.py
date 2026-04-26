"""Text-mode analysis. Prints to stdout; no file output.

Pure reporting layer — all math lives in `pingscraper.stats`. This module
only shapes strings. Each section is a small function with early exits so
the top-level `run()` reads as a linear pipeline, not as nested branches.
"""

from __future__ import annotations

import statistics
from collections import defaultdict
from pathlib import Path

from pingscraper.stats import (
    OUTAGE_MIN_CONSECUTIVE,
    compute_summary,
    find_target_outages,
    load_jsonl,
    parse_pings,
    parse_ts,
)

HEADER_RULE = "=" * 72
SECTION_RULE = "-" * 86


def run(log_dir: Path = Path("logs")) -> int:
    pings, wifi = _load(log_dir)
    if not pings:
        return 1

    summary = compute_summary(pings)
    _print_header(summary)
    _print_target_table(summary)
    _print_outage_bursts(pings, summary)
    _print_hourly_breakdown(pings)
    _print_signal_correlation(pings, wifi)
    print("\n" + HEADER_RULE)
    return 0


# --------------------------------------------------------------------------
# Load
# --------------------------------------------------------------------------


def _load(log_dir: Path) -> tuple[list[dict], list[dict]]:
    ping_files = sorted(log_dir.glob("pings-*.jsonl"))
    if not ping_files:
        print(f"No ping logs found in {log_dir}")
        return [], []

    pings: list[dict] = []
    for pf in ping_files:
        pings.extend(load_jsonl(pf))
    if not pings:
        print("Ping log files found but empty.")
        return [], []

    parse_pings(pings)

    wifi: list[dict] = []
    for wf in sorted(log_dir.glob("wifi-*.jsonl")):
        wifi.extend(load_jsonl(wf))
    return pings, wifi


# --------------------------------------------------------------------------
# Sections
# --------------------------------------------------------------------------


def _print_header(summary: dict) -> None:
    print(HEADER_RULE)
    print("WI-FI MONITOR REPORT")
    print(f"Window: {summary['window_start_utc']}  ->  {summary['window_end_utc']}")
    print(
        f"Duration: {summary['duration_hours']} hours   |   "
        f"Total probes: {summary['total_pings']:,}"
    )
    print(HEADER_RULE)


def _print_target_table(summary: dict) -> None:
    print(
        f"\n{'Target':<22}{'Kind':<10}{'Count':>9}{'Uptime':>9}"
        f"{'p50':>9}{'p95':>9}{'p99':>9}{'jitter':>9}"
    )
    print(SECTION_RULE)
    for label, t in summary["targets"].items():
        print(_format_target_row(label, t))


def _format_target_row(label: str, t: dict) -> str:
    base = (
        f"{label:<22}{t['kind']:<10}{t['total_pings']:>9,}{t['uptime_pct']:>8.2f}%"
    )
    if t["p50_ms"] is None:
        return base + "   (no successful probes)"

    jitter = t["jitter_ms"]
    if jitter is None:
        jitter = 0.0
    return (
        f"{base}{t['p50_ms']:>9.1f}{t['p95_ms']:>9.1f}"
        f"{t['p99_ms']:>9.1f}{jitter:>9.1f}"
    )


def _print_outage_bursts(pings: list[dict], summary: dict) -> None:
    print(f"\nOUTAGE BURSTS (>= {OUTAGE_MIN_CONSECUTIVE} consecutive failures, per target)")
    print(SECTION_RULE)
    for label in summary["targets"]:
        bursts = find_target_outages(pings, label)
        _print_bursts_for_label(label, bursts)


def _print_bursts_for_label(label: str, bursts: list[dict]) -> None:
    print(f"\n  {label}: {len(bursts)} burst(s)")
    for i, b in enumerate(bursts[:25], start=1):
        reason_str = ", ".join(f"{k}={v}" for k, v in b["errors"].items())
        print(
            f"    {i:>3}. {b['start_local']}  ({b['duration_sec']}s)  "
            f"{b['consecutive_cycles_failed']} fails  [{reason_str}]"
        )
    if len(bursts) > 25:
        print(f"    ... and {len(bursts) - 25} more")


def _print_hourly_breakdown(pings: list[dict]) -> None:
    print("\nHOURLY BREAKDOWN (local time, all targets combined)")
    print(SECTION_RULE)
    hourly_total, hourly_fail = _hourly_counts(pings)
    if not hourly_total:
        print("  (no data)")
        return

    print(f"  {'Hour':<20}{'Total':>10}{'Fails':>10}{'Fail %':>10}")
    worst = sorted(
        hourly_total.keys(),
        key=lambda h: hourly_fail[h] / hourly_total[h],
        reverse=True,
    )[:15]
    for h in sorted(worst):
        total = hourly_total[h]
        fails = hourly_fail[h]
        pct = (fails / total) * 100
        bar = "#" * int(pct / 2)
        print(f"  {h:<20}{total:>10}{fails:>10}{pct:>9.1f}% {bar}")


def _hourly_counts(pings: list[dict]) -> tuple[dict[str, int], dict[str, int]]:
    hourly_total: dict[str, int] = defaultdict(int)
    hourly_fail: dict[str, int] = defaultdict(int)
    for p in pings:
        h = p["_ts"].astimezone().strftime("%Y-%m-%d %H")
        hourly_total[h] += 1
        if not p["success"]:
            hourly_fail[h] += 1
    return hourly_total, hourly_fail


def _print_signal_correlation(pings: list[dict], wifi: list[dict]) -> None:
    if not wifi:
        return
    for w in wifi:
        w["_ts"] = parse_ts(w["ts"])
    wifi.sort(key=lambda r: r["_ts"])

    ok_signals, fail_signals = _collect_signals_at_ping_times(pings, wifi)
    if not ok_signals or not fail_signals:
        return

    print("\nSIGNAL STRENGTH AT PROBE TIME")
    print(SECTION_RULE)
    ok_avg = statistics.mean(ok_signals)
    fail_avg = statistics.mean(fail_signals)
    print(
        f"  When probes SUCCEED:  avg {ok_avg:.1f}%  "
        f"(median {statistics.median(ok_signals):.0f}%)"
    )
    print(
        f"  When probes FAIL:     avg {fail_avg:.1f}%  "
        f"(median {statistics.median(fail_signals):.0f}%)"
    )
    diff = ok_avg - fail_avg
    if abs(diff) < 5:
        print(
            f"  -> Signal is similar during failures ({diff:+.1f}%). "
            "Drops are likely upstream, not RF."
        )
    else:
        print(f"  -> Signal drops by {diff:.1f}% during failures. Suggests RF/coverage issue.")


def _collect_signals_at_ping_times(
    pings: list[dict], wifi: list[dict]
) -> tuple[list[int], list[int]]:
    """For each ping, find the nearest-preceding Wi-Fi sample and bucket its
    signal% into ok or fail lists. Single pass — both lists populated together."""
    ok_signals: list[int] = []
    fail_signals: list[int] = []
    idx = 0
    for p in pings:
        while idx + 1 < len(wifi) and wifi[idx + 1]["_ts"] <= p["_ts"]:
            idx += 1
        sig = wifi[idx].get("signal_percent")
        if sig is None:
            continue
        if p["success"]:
            ok_signals.append(sig)
        else:
            fail_signals.append(sig)
    return ok_signals, fail_signals
