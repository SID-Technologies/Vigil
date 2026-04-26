"""Command-line entry point for pingscraper.

Subcommands:
    monitor   Continuously ping router + external targets, write JSONL logs.
    analyze   Print a text summary of collected logs (uptime, latency, outages).
    report    Generate CSV/JSON/HTML reports from collected logs.
"""

from __future__ import annotations

import argparse
from pathlib import Path

from pingscraper import __version__


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="pingscraper",
        description="Continuous Wi-Fi monitor and reporting tool.",
    )
    parser.add_argument(
        "--version",
        action="version",
        version=f"%(prog)s {__version__}",
    )

    sub = parser.add_subparsers(dest="command", required=True, metavar="<command>")

    # monitor ---------------------------------------------------------------
    p_mon = sub.add_parser(
        "monitor",
        help="Run the continuous Wi-Fi monitor.",
        description="Ping the default gateway and external targets at a fixed "
        "interval, sample Wi-Fi signal, and flush JSONL logs to disk.",
    )
    p_mon.add_argument(
        "--log-dir",
        type=Path,
        default=Path("logs"),
        help="Directory for JSONL logs (default: %(default)s).",
    )
    p_mon.add_argument(
        "--interval",
        type=float,
        default=2.5,
        help="Seconds between ping cycles (default: %(default)s).",
    )
    p_mon.add_argument(
        "--flush-interval",
        type=int,
        default=60,
        help="Seconds between disk flushes (default: %(default)s).",
    )
    p_mon.add_argument(
        "--timeout-ms",
        type=int,
        default=2000,
        help="Per-ping timeout in milliseconds (default: %(default)s).",
    )

    # analyze ---------------------------------------------------------------
    p_ana = sub.add_parser(
        "analyze",
        help="Print a text summary of collected logs.",
        description="Read logs and print uptime, latency percentiles, outage "
        "windows, hourly breakdown, and signal-correlation stats.",
    )
    p_ana.add_argument(
        "--log-dir",
        type=Path,
        default=Path("logs"),
        help="Directory to read logs from (default: %(default)s).",
    )

    # report ----------------------------------------------------------------
    p_rep = sub.add_parser(
        "report",
        help="Generate CSV/JSON/HTML reports.",
        description="Read logs and produce wifi-report.{csv,json,html} suitable "
        "for sharing with technicians or non-technical stakeholders.",
    )
    p_rep.add_argument(
        "--log-dir",
        type=Path,
        default=Path("logs"),
        help="Directory to read logs from (default: %(default)s).",
    )
    p_rep.add_argument(
        "--out-dir",
        type=Path,
        default=Path("reports"),
        help="Directory to write reports to (default: %(default)s).",
    )

    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)

    if args.command == "monitor":
        from pingscraper import monitor

        return monitor.run(
            log_dir=args.log_dir,
            ping_interval_sec=args.interval,
            flush_interval_sec=args.flush_interval,
            ping_timeout_ms=args.timeout_ms,
        )

    if args.command == "analyze":
        from pingscraper import analyze

        return analyze.run(log_dir=args.log_dir)

    if args.command == "report":
        from pingscraper import report

        return report.run(log_dir=args.log_dir, out_dir=args.out_dir)

    return 2
