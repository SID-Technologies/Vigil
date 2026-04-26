# pingscraper

Continuous Wi-Fi monitor and reporting CLI for Windows.

Pings the local router and external anycast DNS targets at a fixed interval,
samples Wi-Fi signal strength, and emits JSONL logs plus shareable HTML/CSV
reports. Stdlib-only at runtime.

## Install

```powershell
uv sync
```

This creates `.venv\`, installs the package in editable mode, and pulls in the
dev tooling (`black`, `isort`, `ruff`).

## Usage

```powershell
# Run the monitor (Ctrl+C for clean shutdown)
uv run pingscraper monitor

# Print a text summary
uv run pingscraper analyze

# Generate CSV/JSON/HTML reports in ./reports/
uv run pingscraper report
```

Each subcommand accepts `--log-dir` (default `./logs`). `monitor` also accepts
`--interval`, `--flush-interval`, `--timeout-ms`. `report` accepts `--out-dir`
(default `./reports`). `pingscraper --help` for details.

## Development

```powershell
uv run black src
uv run isort src
uv run ruff check src
```
