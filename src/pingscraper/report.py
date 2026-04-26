"""Report orchestrator.

Loads JSONL logs, runs the shared stats pipeline, and writes three artifacts
into `out_dir`:

    wifi-report.csv    one row per probe (spreadsheet-friendly)
    wifi-report.json   consolidated summary + raw probes + outages
    wifi-report.html   Jinja-rendered dashboard (Chart.js)

The HTML template lives in `pingscraper/templates/report.html.j2` so the
browser-side code can be read and edited without wading through Python.
"""

from __future__ import annotations

import csv
import json
from pathlib import Path

from jinja2 import Environment, PackageLoader, select_autoescape

from pingscraper.stats import (
    bucket_by_hour,
    compute_summary,
    find_full_network_outages,
    find_target_outages,
    load_jsonl,
    parse_pings,
    parse_ts,
)


class Reporter:
    """Owns one run of report generation. Built once, called once.

    A class (rather than a free function) because loading the logs once and
    threading the result through three writers is cleaner than passing the
    data explicitly five times."""

    def __init__(self, log_dir: Path, out_dir: Path) -> None:
        self.log_dir = log_dir
        self.out_dir = out_dir
        self._pings: list[dict] = []
        self._wifi: list[dict] = []
        self._summary: dict = {}
        self._outages: list[dict] = []
        self._target_outages: dict[str, list[dict]] = {}

    # ----------------------------------------------------------------------
    # Public entry point
    # ----------------------------------------------------------------------

    def run(self) -> int:
        if not self.log_dir.exists():
            print(f"Log directory not found: {self.log_dir}")
            return 1

        if not self._load():
            return 1

        self._analyze()
        self.out_dir.mkdir(parents=True, exist_ok=True)

        csv_path = self.out_dir / "wifi-report.csv"
        json_path = self.out_dir / "wifi-report.json"
        html_path = self.out_dir / "wifi-report.html"

        print(f"Writing {csv_path}...")
        self._write_csv(csv_path)

        print(f"Writing {json_path}...")
        self._write_json(json_path)

        print(f"Writing {html_path}...")
        self._write_html(html_path)

        self._print_summary(html_path)
        return 0

    # ----------------------------------------------------------------------
    # Load + analyze
    # ----------------------------------------------------------------------

    def _load(self) -> bool:
        ping_files = sorted(self.log_dir.glob("pings-*.jsonl"))
        if not ping_files:
            print(f"No ping logs found in {self.log_dir}")
            return False

        for path in ping_files:
            self._pings.extend(load_jsonl(path))
        if not self._pings:
            print("Ping log files found but empty.")
            return False

        parse_pings(self._pings)

        for path in sorted(self.log_dir.glob("wifi-*.jsonl")):
            self._wifi.extend(load_jsonl(path))
        for w in self._wifi:
            w["_ts"] = parse_ts(w["ts"])

        print(f"Loaded {len(self._pings):,} probes and {len(self._wifi)} Wi-Fi samples.")
        return True

    def _analyze(self) -> None:
        print("Computing summary...")
        self._summary = compute_summary(self._pings)
        self._outages = find_full_network_outages(self._pings)
        self._target_outages = {
            label: find_target_outages(self._pings, label)
            for label in self._summary["targets"]
        }

    # ----------------------------------------------------------------------
    # Writers
    # ----------------------------------------------------------------------

    def _write_csv(self, path: Path) -> None:
        with path.open("w", encoding="utf-8", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(_CSV_HEADER)
            for p in self._pings:
                writer.writerow(_ping_to_csv_row(p))

    def _write_json(self, path: Path) -> None:
        summary_payload = dict(self._summary)
        summary_payload["outages"] = self._outages
        summary_payload["target_outages"] = self._target_outages

        payload = {
            "summary": summary_payload,
            "pings": [_strip_internal(p) for p in self._pings],
            "wifi_samples": [_strip_internal(w) for w in self._wifi],
        }
        with path.open("w", encoding="utf-8") as f:
            json.dump(payload, f, indent=2)

    def _write_html(self, path: Path) -> None:
        data = {
            "summary": self._summary,
            "hourly": bucket_by_hour(self._pings),
            "outages": self._outages,
            "target_outages": self._target_outages,
        }
        html = _render_html_template(data)
        path.write_text(html, encoding="utf-8")

    def _print_summary(self, html_path: Path) -> None:
        total_bursts = sum(len(v) for v in self._target_outages.values())
        print()
        print(f"Done. Open {html_path} in your browser to see the charts.")
        print()
        print("Summary:")
        for label, t in self._summary["targets"].items():
            line = _format_summary_line(label, t, len(self._target_outages.get(label, [])))
            print("  " + line)
        print(
            f"  {len(self._outages)} full-network outage(s), "
            f"{total_bursts} per-target outage burst(s)"
        )


# --------------------------------------------------------------------------
# Module-level helpers (pure, no state)
# --------------------------------------------------------------------------


_CSV_HEADER = [
    "timestamp_utc",
    "timestamp_local",
    "target_label",
    "target_kind",
    "target_host",
    "target_port",
    "success",
    "rtt_ms",
    "error",
]


def _ping_to_csv_row(p: dict) -> list:
    ts_utc = parse_ts(p["ts"])
    success_str = "no"
    if p["success"]:
        success_str = "yes"
    rtt = ""
    if p["rtt_ms"] is not None:
        rtt = p["rtt_ms"]
    return [
        ts_utc.isoformat(timespec="seconds"),
        ts_utc.astimezone().strftime("%Y-%m-%d %H:%M:%S"),
        p["target_label"],
        p.get("target_kind", "icmp"),
        p.get("target_host", ""),
        p.get("target_port", "") or "",
        success_str,
        rtt,
        p.get("error") or "",
    ]


def _strip_internal(record: dict) -> dict:
    """Drop keys starting with `_` (e.g. the `_ts` datetime attached at load time)."""
    out: dict = {}
    for k, v in record.items():
        if k.startswith("_"):
            continue
        out[k] = v
    return out


def _format_summary_line(label: str, t: dict, burst_count: int) -> str:
    jitter = t["jitter_ms"]
    if jitter is None:
        jitter = "-"
    burst_word = "bursts"
    if burst_count == 1:
        burst_word = "burst"
    return (
        f"{label:<22} uptime {t['uptime_pct']:>6.2f}%  "
        f"(p50 {t['p50_ms']}ms, p99 {t['p99_ms']}ms, "
        f"jitter {jitter}ms, {burst_count} {burst_word})"
    )


_JINJA_ENV: Environment | None = None


def _jinja_env() -> Environment:
    """Lazy-built so report generation is cheap if the template is unchanged."""
    global _JINJA_ENV
    if _JINJA_ENV is None:
        _JINJA_ENV = Environment(
            loader=PackageLoader("pingscraper", "templates"),
            autoescape=select_autoescape(disabled_extensions=("j2",)),
            trim_blocks=True,
            lstrip_blocks=True,
        )
    return _JINJA_ENV


def _render_html_template(data: dict) -> str:
    template = _jinja_env().get_template("report.html.j2")
    return template.render(data=data)


# --------------------------------------------------------------------------
# CLI entry point — matches the old free-function API the CLI dispatches to.
# --------------------------------------------------------------------------


def run(log_dir: Path = Path("logs"), out_dir: Path = Path("reports")) -> int:
    return Reporter(log_dir=log_dir, out_dir=out_dir).run()


__all__ = ["Reporter", "run"]
