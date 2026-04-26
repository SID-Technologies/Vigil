"""Probe hierarchy.

Each probe kind is its own class so dispatch is polymorphic (no if-chain), each
kind is testable in isolation (tests mock sockets or subprocess per class), and
a new kind is added by subclassing — not by editing a central dispatcher.

A probe holds its `Target` and exposes `run(timeout_ms) -> PingResult`. The
subclass only implements `_execute` which returns the tuple `(success, rtt_ms,
error)`. Everything else (timestamp, packaging into `PingResult`) is shared in
the base class so subclasses cannot drift.

Probe choice by target kind:
    icmp      — Windows `ping -n 1`. Network-layer reachability.
    tcp       — `socket.connect((host, port))` handshake time. Proves the
                actual service port is open.
    udp_dns   — real DNS query over UDP to a public resolver.
    udp_stun  — RFC 5389 STUN Binding Request; the exact UDP call-plane
                protocol Teams / Zoom / every WebRTC client uses at call
                setup. The closest DIY proxy for real call quality.
"""

from __future__ import annotations

import contextlib
import os
import re
import socket
import struct
import subprocess
import time
from abc import ABC, abstractmethod

from pingscraper.models import PingResult, Target, iso_now

# --------------------------------------------------------------------------
# Base class
# --------------------------------------------------------------------------


class Probe(ABC):
    """Base probe. Subclasses set `kind` and implement `_execute`."""

    kind: str = ""

    def __init__(self, target: Target) -> None:
        if target.kind != self.kind:
            raise ValueError(
                f"{type(self).__name__} requires kind={self.kind!r}, got {target.kind!r}"
            )
        self.target = target

    def run(self, timeout_ms: int) -> PingResult:
        ts = iso_now()
        ok, rtt, err = self._execute(timeout_ms)
        return PingResult(
            ts=ts,
            target_label=self.target.label,
            target_host=self.target.host,
            target_port=self.target.port,
            target_kind=self.kind,
            success=ok,
            rtt_ms=rtt,
            error=err,
        )

    @abstractmethod
    def _execute(self, timeout_ms: int) -> tuple[bool, float | None, str | None]:
        """Fire the probe once. Return (success, rtt_ms, error_label)."""


# --------------------------------------------------------------------------
# ICMP
# --------------------------------------------------------------------------

_PING_RTT_RE = re.compile(r"time[=<]\s*([\d.]+)\s*ms", re.IGNORECASE)


class IcmpProbe(Probe):
    kind = "icmp"

    def _execute(self, timeout_ms: int) -> tuple[bool, float | None, str | None]:
        host = self.target.host
        try:
            proc = subprocess.run(
                ["ping", "-n", "1", "-w", str(timeout_ms), host],
                capture_output=True,
                text=True,
                timeout=(timeout_ms / 1000) + 2,
                check=False,
            )
        except subprocess.TimeoutExpired:
            return False, None, "process_timeout"
        except FileNotFoundError:
            return False, None, "ping_not_found"

        out = (proc.stdout or "") + (proc.stderr or "")

        if proc.returncode == 0 and "TTL=" in out:
            m = _PING_RTT_RE.search(out)
            if m is None:
                return True, 0.0, None
            return True, float(m.group(1)), None

        return False, None, _classify_ping_error(out, proc.returncode)


def _classify_ping_error(stdout_stderr: str, returncode: int) -> str:
    """Early-exit classifier for Windows `ping` failure modes."""
    low = stdout_stderr.lower()
    if "timed out" in low:
        return "timeout"
    if "destination host unreachable" in low:
        return "host_unreachable"
    if "destination net unreachable" in low:
        return "net_unreachable"
    if "could not find host" in low or "ping request could not find" in low:
        return "dns"
    if "transmit failed" in low:
        return "transmit_failed"
    return f"unknown(rc={returncode})"


# --------------------------------------------------------------------------
# TCP handshake
# --------------------------------------------------------------------------


class TcpProbe(Probe):
    kind = "tcp"

    def _execute(self, timeout_ms: int) -> tuple[bool, float | None, str | None]:
        if self.target.port is None:
            return False, None, "missing_port"
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(timeout_ms / 1000)
        try:
            start = time.monotonic()
            sock.connect((self.target.host, self.target.port))
            return True, round((time.monotonic() - start) * 1000, 2), None
        except TimeoutError:
            return False, None, "timeout"
        except ConnectionRefusedError:
            return False, None, "conn_refused"
        except socket.gaierror:
            return False, None, "dns"
        except OSError as e:
            return False, None, f"oserror:{e.errno}"
        finally:
            with contextlib.suppress(OSError):
                sock.close()


# --------------------------------------------------------------------------
# UDP — DNS query
# --------------------------------------------------------------------------


def _dns_query_packet(tid: bytes) -> bytes:
    # RFC 1035 header + question for "example.com IN A". Minimal and cached
    # upstream almost everywhere, so the query itself is cheap.
    return (
        tid
        + b"\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00"  # flags + section counts
        + b"\x07example\x03com\x00\x00\x01\x00\x01"  # qname + type A + class IN
    )


class DnsUdpProbe(Probe):
    kind = "udp_dns"

    def _execute(self, timeout_ms: int) -> tuple[bool, float | None, str | None]:
        port = self.target.port or 53
        tid = os.urandom(2)
        packet = _dns_query_packet(tid)
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(timeout_ms / 1000)
        try:
            start = time.monotonic()
            sock.sendto(packet, (self.target.host, port))
            data, _ = sock.recvfrom(512)
            rtt = round((time.monotonic() - start) * 1000, 2)
            if len(data) < 2 or data[:2] != tid:
                return False, None, "tid_mismatch"
            return True, rtt, None
        except TimeoutError:
            return False, None, "timeout"
        except socket.gaierror:
            return False, None, "dns"
        except OSError as e:
            return False, None, f"oserror:{e.errno}"
        finally:
            with contextlib.suppress(OSError):
                sock.close()


# --------------------------------------------------------------------------
# UDP — STUN Binding Request
# --------------------------------------------------------------------------

_STUN_MAGIC_COOKIE = b"\x21\x12\xa4\x42"


def _stun_binding_request(trans_id: bytes) -> bytes:
    # Header: type=0x0001 (Binding Request), length=0, magic cookie, trans_id.
    return struct.pack(">HH", 0x0001, 0) + _STUN_MAGIC_COOKIE + trans_id


def _is_valid_stun_response(data: bytes, trans_id: bytes) -> bool:
    if len(data) < 20:
        return False
    if data[4:8] != _STUN_MAGIC_COOKIE:
        return False
    return data[8:20] == trans_id


class StunUdpProbe(Probe):
    kind = "udp_stun"

    def _execute(self, timeout_ms: int) -> tuple[bool, float | None, str | None]:
        port = self.target.port or 3478
        trans_id = os.urandom(12)
        packet = _stun_binding_request(trans_id)
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(timeout_ms / 1000)
        try:
            start = time.monotonic()
            sock.sendto(packet, (self.target.host, port))
            data, _ = sock.recvfrom(1024)
            rtt = round((time.monotonic() - start) * 1000, 2)
            if not _is_valid_stun_response(data, trans_id):
                return False, None, "malformed_response"
            return True, rtt, None
        except TimeoutError:
            return False, None, "timeout"
        except socket.gaierror:
            return False, None, "dns"
        except OSError as e:
            return False, None, f"oserror:{e.errno}"
        finally:
            with contextlib.suppress(OSError):
                sock.close()


# --------------------------------------------------------------------------
# Factory + default targets
# --------------------------------------------------------------------------

_PROBE_CLASSES: dict[str, type[Probe]] = {
    IcmpProbe.kind: IcmpProbe,
    TcpProbe.kind: TcpProbe,
    DnsUdpProbe.kind: DnsUdpProbe,
    StunUdpProbe.kind: StunUdpProbe,
}


def build_probe(target: Target) -> Probe:
    cls = _PROBE_CLASSES.get(target.kind)
    if cls is None:
        raise ValueError(f"Unknown probe kind: {target.kind!r}")
    return cls(target)


# The service names and kinds are chosen so a hostile stakeholder ("it's not
# our fault, that's just Google") cannot hand-wave the evidence away: the
# probes hit the actual Teams / Zoom / Outlook endpoints, and the STUN probes
# exercise the exact UDP protocol those services use at call setup.
DEFAULT_TARGETS: list[Target] = [
    # ICMP — network-layer reachability to both generic anycast and the real
    # video-call hostnames.
    Target("google_dns_icmp", "icmp", "8.8.8.8"),
    Target("cloudflare_dns_icmp", "icmp", "1.1.1.1"),
    Target("teams_icmp", "icmp", "teams.microsoft.com"),
    Target("zoom_icmp", "icmp", "zoom.us"),
    Target("outlook_icmp", "icmp", "outlook.office.com"),
    # TCP :443 — some ISPs drop HTTPS while leaving ICMP alone, or vice versa.
    Target("teams_tcp443", "tcp", "teams.microsoft.com", 443),
    Target("zoom_tcp443", "tcp", "zoom.us", 443),
    Target("outlook_tcp443", "tcp", "outlook.office.com", 443),
    # UDP DNS — real UDP traffic to well-known public resolvers on :53.
    Target("google_dns_udp", "udp_dns", "8.8.8.8", 53),
    Target("cloudflare_dns_udp", "udp_dns", "1.1.1.1", 53),
    # UDP STUN — WebRTC / Teams / Zoom call-plane protocol.
    Target("google_stun_udp", "udp_stun", "stun.l.google.com", 19302),
    Target("cloudflare_stun_udp", "udp_stun", "stun.cloudflare.com", 3478),
]
