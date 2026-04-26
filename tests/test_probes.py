"""Unit tests for the probe classes.

Socket-level tests focus on the pieces that are most likely to go wrong
silently in production: protocol parsing (STUN magic cookie + transaction id
echo, DNS transaction id match) and error classification for ICMP.

End-to-end wire tests are covered by the smoke run in `make test-smoke`,
not here.
"""

from __future__ import annotations

import os
import struct
from unittest.mock import MagicMock, patch

import pytest

from pingscraper.models import Target
from pingscraper.probes import (
    DnsUdpProbe,
    IcmpProbe,
    StunUdpProbe,
    TcpProbe,
    _classify_ping_error,
    _dns_query_packet,
    _is_valid_stun_response,
    _stun_binding_request,
    build_probe,
)

STUN_MAGIC_COOKIE = b"\x21\x12\xa4\x42"


# --------------------------------------------------------------------------
# Factory
# --------------------------------------------------------------------------


def test_build_probe_returns_right_class_for_each_kind():
    cases = [
        ("icmp", IcmpProbe),
        ("tcp", TcpProbe),
        ("udp_dns", DnsUdpProbe),
        ("udp_stun", StunUdpProbe),
    ]
    for kind, expected_cls in cases:
        target = Target("lbl", kind, "example.com", 443 if kind != "icmp" else None)
        probe = build_probe(target)
        assert isinstance(probe, expected_cls)
        assert probe.target is target


def test_build_probe_rejects_unknown_kind():
    target = Target("lbl", "carrier_pigeon", "example.com")
    with pytest.raises(ValueError, match="Unknown probe kind"):
        build_probe(target)


def test_probe_constructor_rejects_kind_mismatch():
    target = Target("lbl", "tcp", "example.com", 443)
    with pytest.raises(ValueError, match="requires kind='icmp'"):
        IcmpProbe(target)


# --------------------------------------------------------------------------
# ICMP — protocol parsing via subprocess mock
# --------------------------------------------------------------------------


def _mock_ping_result(returncode: int, stdout: str = "", stderr: str = ""):
    proc = MagicMock()
    proc.returncode = returncode
    proc.stdout = stdout
    proc.stderr = stderr
    return proc


def test_icmp_probe_parses_rtt_on_success():
    target = Target("lbl", "icmp", "8.8.8.8")
    probe = IcmpProbe(target)
    fake_stdout = (
        "Pinging 8.8.8.8 with 32 bytes of data:\r\n"
        "Reply from 8.8.8.8: bytes=32 time=17ms TTL=118\r\n"
    )
    with patch("pingscraper.probes.subprocess.run", return_value=_mock_ping_result(0, fake_stdout)):
        result = probe.run(timeout_ms=2000)

    assert result.success is True
    assert result.rtt_ms == 17.0
    assert result.error is None
    assert result.target_kind == "icmp"


def test_icmp_probe_classifies_timeout():
    target = Target("lbl", "icmp", "8.8.8.8")
    probe = IcmpProbe(target)
    fake_stdout = "Request timed out.\r\n"
    with patch("pingscraper.probes.subprocess.run", return_value=_mock_ping_result(1, fake_stdout)):
        result = probe.run(timeout_ms=2000)

    assert result.success is False
    assert result.error == "timeout"


def test_classify_ping_error_handles_known_variants():
    assert _classify_ping_error("Request timed out.", 1) == "timeout"
    assert _classify_ping_error("Destination host unreachable.", 1) == "host_unreachable"
    assert _classify_ping_error("Destination net unreachable.", 1) == "net_unreachable"
    assert _classify_ping_error("Ping request could not find host", 1) == "dns"
    assert _classify_ping_error("General failure.", 99).startswith("unknown(")


# --------------------------------------------------------------------------
# DNS UDP — packet crafting
# --------------------------------------------------------------------------


def test_dns_query_packet_embeds_tid():
    tid = b"\xab\xcd"
    packet = _dns_query_packet(tid)
    assert packet.startswith(tid)
    # Flags: standard query, recursion desired (0x0100)
    assert packet[2:4] == b"\x01\x00"
    # QDCOUNT = 1, ANCOUNT = NSCOUNT = ARCOUNT = 0
    assert packet[4:12] == b"\x00\x01\x00\x00\x00\x00\x00\x00"


# --------------------------------------------------------------------------
# STUN UDP — packet crafting + response validation
# --------------------------------------------------------------------------


def test_stun_binding_request_has_magic_cookie_and_trans_id():
    trans_id = os.urandom(12)
    packet = _stun_binding_request(trans_id)

    assert len(packet) == 20
    msg_type, msg_len = struct.unpack(">HH", packet[0:4])
    assert msg_type == 0x0001  # Binding Request
    assert msg_len == 0
    assert packet[4:8] == STUN_MAGIC_COOKIE
    assert packet[8:20] == trans_id


def test_is_valid_stun_response_accepts_well_formed():
    trans_id = os.urandom(12)
    # A plausible Binding Success Response: type=0x0101, len=0, cookie, trans_id.
    header = struct.pack(">HH", 0x0101, 0) + STUN_MAGIC_COOKIE + trans_id
    assert _is_valid_stun_response(header, trans_id) is True


def test_is_valid_stun_response_rejects_short_packet():
    assert _is_valid_stun_response(b"\x00" * 10, os.urandom(12)) is False


def test_is_valid_stun_response_rejects_wrong_magic_cookie():
    trans_id = os.urandom(12)
    bad = struct.pack(">HH", 0x0101, 0) + b"\xff\xff\xff\xff" + trans_id
    assert _is_valid_stun_response(bad, trans_id) is False


def test_is_valid_stun_response_rejects_mismatched_trans_id():
    trans_id = os.urandom(12)
    other = os.urandom(12)
    assert other != trans_id
    response = struct.pack(">HH", 0x0101, 0) + STUN_MAGIC_COOKIE + other
    assert _is_valid_stun_response(response, trans_id) is False


# --------------------------------------------------------------------------
# TCP handshake probe — mocked socket
# --------------------------------------------------------------------------


def test_tcp_probe_returns_success_and_nonzero_rtt():
    target = Target("lbl", "tcp", "example.com", 443)
    probe = TcpProbe(target)

    with patch("pingscraper.probes.socket.socket") as sock_cls:
        sock = sock_cls.return_value
        sock.connect.return_value = None
        result = probe.run(timeout_ms=2000)

    assert result.success is True
    assert result.rtt_ms is not None
    assert result.rtt_ms >= 0
    assert result.target_port == 443


def test_tcp_probe_classifies_conn_refused():
    target = Target("lbl", "tcp", "example.com", 12345)
    probe = TcpProbe(target)

    with patch("pingscraper.probes.socket.socket") as sock_cls:
        sock = sock_cls.return_value
        sock.connect.side_effect = ConnectionRefusedError()
        result = probe.run(timeout_ms=2000)

    assert result.success is False
    assert result.error == "conn_refused"
