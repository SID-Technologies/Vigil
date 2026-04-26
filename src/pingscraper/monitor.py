"""Continuous Wi-Fi monitor.

`Monitor` owns the run loop: build probes, fire them in parallel every cycle,
buffer results, and flush to daily JSONL files. Free functions below handle
the Windows-specific bits (`netsh`, `route print`) so they stay testable and
swappable on other platforms later.

Output (relative to `log_dir`):
    pings-YYYY-MM-DD.jsonl    one line per probe
    wifi-YYYY-MM-DD.jsonl     one line per flush (SSID, signal, BSSID, rates)
    monitor.log               human-readable progress
"""

from __future__ import annotations

import contextlib
import json
import logging
import re
import signal
import subprocess
import sys
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from dataclasses import asdict
from pathlib import Path

from pingscraper.models import PingResult, WifiSample, iso_now
from pingscraper.probes import DEFAULT_TARGETS, Probe, Target, build_probe

log = logging.getLogger("pingscraper.monitor")


# --------------------------------------------------------------------------
# Windows-specific system helpers (free functions — trivially swappable).
# --------------------------------------------------------------------------


def detect_default_gateway() -> str | None:
    """Return the default-route gateway IP, or None if we can't find it."""
    try:
        proc = subprocess.run(
            ["route", "print", "0.0.0.0"],
            capture_output=True,
            text=True,
            timeout=5,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        return None
    for line in proc.stdout.splitlines():
        parts = line.split()
        if len(parts) >= 3 and parts[0] == "0.0.0.0" and parts[1] == "0.0.0.0":
            return parts[2]
    return None


def sample_wifi() -> WifiSample:
    """Parse `netsh wlan show interfaces`. Returns a mostly-empty sample on
    wired connections or on any parse error — never raises."""
    ts = iso_now()
    try:
        proc = subprocess.run(
            ["netsh", "wlan", "show", "interfaces"],
            capture_output=True,
            text=True,
            timeout=5,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        return WifiSample(ts, None, None, None, None, None, None, None)

    return _parse_netsh_wlan(ts, proc.stdout)


def _parse_netsh_wlan(ts: str, stdout: str) -> WifiSample:
    def grab(key: str) -> str | None:
        m = re.search(rf"^\s*{re.escape(key)}\s*:\s*(.+?)\s*$", stdout, re.MULTILINE)
        return m.group(1) if m else None

    signal_pct, rssi_est = _parse_signal(grab("Signal"))
    return WifiSample(
        ts=ts,
        ssid=grab("SSID"),
        bssid=grab("BSSID"),
        signal_percent=signal_pct,
        rssi_dbm_estimate=rssi_est,
        rx_rate_mbps=_to_float(grab("Receive rate (Mbps)")),
        tx_rate_mbps=_to_float(grab("Transmit rate (Mbps)")),
        channel=grab("Channel"),
    )


def _parse_signal(raw: str | None) -> tuple[int | None, int | None]:
    if raw is None or not raw.endswith("%"):
        return None, None
    try:
        pct = int(raw.rstrip("%").strip())
    except ValueError:
        return None, None
    # Rough Windows convention: 100% ≈ -50 dBm, 0% ≈ -100 dBm.
    return pct, -100 + (pct // 2)


def _to_float(s: str | None) -> float | None:
    if s is None:
        return None
    try:
        return float(s)
    except ValueError:
        return None


# --------------------------------------------------------------------------
# Disk writer
# --------------------------------------------------------------------------


class DailyJsonlWriter:
    """Appends JSONL records to `<log_dir>/<prefix>-YYYY-MM-DD.jsonl` by UTC date."""

    def __init__(self, log_dir: Path, prefix: str) -> None:
        self.log_dir = log_dir
        self.prefix = prefix
        self.log_dir.mkdir(parents=True, exist_ok=True)

    def write_many(self, records: list[dict]) -> None:
        if not records:
            return
        for date, group in _group_by_date(records).items():
            path = self.log_dir / f"{self.prefix}-{date}.jsonl"
            with path.open("a", encoding="utf-8") as f:
                for r in group:
                    f.write(json.dumps(r, separators=(",", ":")) + "\n")


def _group_by_date(records: list[dict]) -> dict[str, list[dict]]:
    """Group records by the UTC date slice of their `ts` field.
    A flush spanning midnight writes to both days' files."""
    out: dict[str, list[dict]] = {}
    for r in records:
        out.setdefault(r["ts"][:10], []).append(r)
    return out


# --------------------------------------------------------------------------
# Monitor
# --------------------------------------------------------------------------


class Monitor:
    """Runs the probe loop. Instantiate once, call `run()`.

    Blocks until SIGINT / SIGTERM. A clean final flush happens on shutdown so
    buffered probes aren't lost — though a Windows Update-forced reboot can
    still drop up to `flush_interval_sec` of data, which is why the flush
    interval defaults to 60 s (small blast radius) rather than something
    larger for efficiency."""

    def __init__(
        self,
        log_dir: Path = Path("logs"),
        ping_interval_sec: float = 2.5,
        flush_interval_sec: int = 60,
        ping_timeout_ms: int = 2000,
        targets: list[Target] | None = None,
    ) -> None:
        self.log_dir = log_dir
        self.ping_interval_sec = ping_interval_sec
        self.flush_interval_sec = flush_interval_sec
        self.ping_timeout_ms = ping_timeout_ms
        self._configured_targets = targets

        self._buffer: list[dict] = []
        self._buffer_lock = threading.Lock()
        self._stop_event = threading.Event()

    # ----------------------------------------------------------------------
    # Public entry point
    # ----------------------------------------------------------------------

    def run(self) -> int:
        self._configure_logging()
        probes = self._resolve_probes()
        self._log_startup_config(probes)

        ping_writer = DailyJsonlWriter(self.log_dir, "pings")
        wifi_writer = DailyJsonlWriter(self.log_dir, "wifi")

        self._install_signal_handlers()
        flusher = threading.Thread(
            target=self._flush_loop,
            args=(ping_writer, wifi_writer),
            name="flusher",
            daemon=True,
        )
        flusher.start()

        self._probe_loop(probes)
        self._final_flush(ping_writer, wifi_writer)
        return 0

    # ----------------------------------------------------------------------
    # Startup
    # ----------------------------------------------------------------------

    def _configure_logging(self) -> None:
        self.log_dir.mkdir(parents=True, exist_ok=True)
        logging.basicConfig(
            level=logging.INFO,
            format="%(asctime)s %(levelname)s %(message)s",
            handlers=[
                logging.FileHandler(self.log_dir / "monitor.log", encoding="utf-8"),
                logging.StreamHandler(sys.stdout),
            ],
        )

    def _resolve_probes(self) -> list[Probe]:
        targets = list(self._configured_targets or DEFAULT_TARGETS)
        gateway = detect_default_gateway()
        if gateway:
            targets.insert(0, Target("router_icmp", "icmp", gateway))
            log.info("Detected default gateway: %s", gateway)
        else:
            log.warning("No default gateway detected — continuing without router probe.")
        return [build_probe(t) for t in targets]

    def _log_startup_config(self, probes: list[Probe]) -> None:
        log.info(
            "Targets: %s",
            ", ".join(_describe_probe(p) for p in probes),
        )
        log.info(
            "Ping interval: %.1fs | Flush interval: %ds | Timeout: %dms",
            self.ping_interval_sec,
            self.flush_interval_sec,
            self.ping_timeout_ms,
        )

    def _install_signal_handlers(self) -> None:
        def handler(_signum, _frame):
            log.info("Shutdown signal received — flushing and exiting.")
            self._stop_event.set()

        signal.signal(signal.SIGINT, handler)
        with contextlib.suppress(AttributeError, ValueError):
            signal.signal(signal.SIGTERM, handler)

    # ----------------------------------------------------------------------
    # Probe loop
    # ----------------------------------------------------------------------

    def _probe_loop(self, probes: list[Probe]) -> None:
        executor = ThreadPoolExecutor(
            max_workers=max(1, len(probes)),
            thread_name_prefix="probe",
        )
        cycle_start = time.monotonic()
        try:
            while not self._stop_event.is_set():
                self._run_one_cycle(executor, probes)
                cycle_start = self._sleep_until_next_cycle(cycle_start)
        finally:
            executor.shutdown(wait=False, cancel_futures=True)

    def _run_one_cycle(
        self, executor: ThreadPoolExecutor, probes: list[Probe]
    ) -> None:
        futures = [executor.submit(p.run, self.ping_timeout_ms) for p in probes]
        results: list[dict] = []
        for f in futures:
            try:
                results.append(asdict(f.result()))
            except Exception:
                log.exception("Probe raised unexpectedly")
        if not results:
            return
        with self._buffer_lock:
            self._buffer.extend(results)

    def _sleep_until_next_cycle(self, cycle_start: float) -> float:
        next_start = cycle_start + self.ping_interval_sec
        sleep_for = next_start - time.monotonic()
        if sleep_for <= 0:
            # Fell behind — resync rather than fire rapidly.
            return time.monotonic()
        self._stop_event.wait(sleep_for)
        return next_start

    # ----------------------------------------------------------------------
    # Flush loop
    # ----------------------------------------------------------------------

    def _flush_loop(
        self, ping_writer: DailyJsonlWriter, wifi_writer: DailyJsonlWriter
    ) -> None:
        while not self._stop_event.is_set():
            self._stop_event.wait(self.flush_interval_sec)
            if self._stop_event.is_set():
                return
            self._flush_once(ping_writer, wifi_writer)

    def _flush_once(
        self, ping_writer: DailyJsonlWriter, wifi_writer: DailyJsonlWriter
    ) -> None:
        wifi = self._sample_wifi_or_log()
        if wifi is not None:
            wifi_writer.write_many([asdict(wifi)])

        with self._buffer_lock:
            to_write = self._buffer[:]
            self._buffer.clear()

        try:
            ping_writer.write_many(to_write)
        except Exception:
            log.exception("Flush failed — re-queuing %d records", len(to_write))
            with self._buffer_lock:
                self._buffer[:0] = to_write
            return

        ok = sum(1 for r in to_write if r["success"])
        log.info(
            "Flushed %d probes (%d ok, %d fail) | ssid=%s signal=%s%%",
            len(to_write),
            ok,
            len(to_write) - ok,
            wifi.ssid if wifi else None,
            wifi.signal_percent if wifi else None,
        )

    @staticmethod
    def _sample_wifi_or_log() -> WifiSample | None:
        try:
            return sample_wifi()
        except Exception:
            log.exception("Wi-Fi sample failed")
            return None

    # ----------------------------------------------------------------------
    # Shutdown
    # ----------------------------------------------------------------------

    def _final_flush(
        self, ping_writer: DailyJsonlWriter, wifi_writer: DailyJsonlWriter
    ) -> None:
        wifi = self._sample_wifi_or_log()
        if wifi is not None:
            with contextlib.suppress(Exception):
                wifi_writer.write_many([asdict(wifi)])

        with self._buffer_lock:
            final = self._buffer[:]
            self._buffer.clear()

        ping_writer.write_many(final)
        log.info("Final flush complete: %d probes. Goodbye.", len(final))


def _describe_probe(p: Probe) -> str:
    suffix = f":{p.target.port}" if p.target.port else ""
    return f"{p.target.label}[{p.kind}]={p.target.host}{suffix}"


# --------------------------------------------------------------------------
# CLI entry point — back-compat shim so `cli.py` keeps calling `monitor.run(...)`
# --------------------------------------------------------------------------


def run(
    log_dir: Path = Path("logs"),
    ping_interval_sec: float = 2.5,
    flush_interval_sec: int = 60,
    ping_timeout_ms: int = 2000,
    targets: list[Target] | None = None,
) -> int:
    """Convenience function that mirrors the old API."""
    return Monitor(
        log_dir=log_dir,
        ping_interval_sec=ping_interval_sec,
        flush_interval_sec=flush_interval_sec,
        ping_timeout_ms=ping_timeout_ms,
        targets=targets,
    ).run()


# Re-export PingResult so existing `from pingscraper.monitor import PingResult`
# continues to work. Kept narrow — new code should import from models directly.
__all__ = [
    "DailyJsonlWriter",
    "Monitor",
    "PingResult",
    "detect_default_gateway",
    "run",
    "sample_wifi",
]
