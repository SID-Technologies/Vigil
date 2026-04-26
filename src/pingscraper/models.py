"""Shared dataclasses.

Kept free of behavior and of imports from other pingscraper modules so every
other module can depend on this one without cycles.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone


def iso_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds")


@dataclass(frozen=True)
class Target:
    """A probe target. `kind` selects the Probe subclass."""

    label: str
    kind: str  # "icmp" | "tcp" | "udp_dns" | "udp_stun"
    host: str
    port: int | None = None


@dataclass
class PingResult:
    ts: str
    target_label: str
    target_host: str
    target_port: int | None
    target_kind: str
    success: bool
    rtt_ms: float | None
    error: str | None


@dataclass
class WifiSample:
    ts: str
    ssid: str | None
    bssid: str | None
    signal_percent: int | None
    rssi_dbm_estimate: int | None
    rx_rate_mbps: float | None
    tx_rate_mbps: float | None
    channel: str | None
