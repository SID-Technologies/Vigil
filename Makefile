# pingscraper — developer commands
#
# Works from PowerShell, cmd.exe, or Git Bash. No bash/awk/find required.
#
# Only `sync`, `lock`, and `upgrade` need `uv` on PATH. Every other target
# uses the .venv binaries directly, so once `make sync` has run once, the
# rest of the Makefile keeps working even if uv falls off PATH.
#
# Pass flags through to the CLI with ARGS, e.g.:
#   make monitor ARGS="--interval 1.0 --log-dir D:/logs"
#   make report  ARGS="--out-dir D:/reports"

.DEFAULT_GOAL := help

UV        ?= uv
VENV_BIN  := .venv/Scripts
CLI       := $(VENV_BIN)/pingscraper.exe
PY        := $(VENV_BIN)/python.exe
BLACK     := $(VENV_BIN)/black.exe
ISORT     := $(VENV_BIN)/isort.exe
RUFF      := $(VENV_BIN)/ruff.exe
PYTEST    := $(VENV_BIN)/pytest.exe
SRC       := src
TESTS     := tests

# Fallback for targets that don't need .venv (help/clean) — system Python.
SYSPY     ?= python

# ---------------------------------------------------------------------------
# Help (default target)
# ---------------------------------------------------------------------------

define HELP_TEXT
usage: make [-h] <target> [ARGS="..."]

Developer commands for pingscraper.

targets:
  environment
    sync        Install project + dev deps into .venv (needs uv on PATH)
    lock        Refresh uv.lock (needs uv on PATH)
    upgrade     Upgrade locked deps to latest allowed versions

  run
    monitor     Start the Wi-Fi monitor (Ctrl+C to stop)
    analyze     Print text analysis of collected logs
    report      Generate CSV/JSON/HTML reports
    version     Print CLI version

  lint & test
    lint        Check with black, isort, ruff
    format      Apply isort + black
    fix         ruff --fix + format
    check       Alias for lint
    test        Run pytest

  clean
    clean       Remove build artifacts and caches
    distclean   clean + remove .venv, logs/, reports/, uv.lock

options:
  -h, --help    Show this help message
endef
export HELP_TEXT

.PHONY: help
help:
	@$(SYSPY) -c "import os; print(os.environ['HELP_TEXT'])"

# ---------------------------------------------------------------------------
# Environment — these need `uv` on PATH
# ---------------------------------------------------------------------------

.PHONY: sync
sync:
	$(UV) sync

.PHONY: lock
lock:
	$(UV) lock

.PHONY: upgrade
upgrade:
	$(UV) lock --upgrade
	$(UV) sync

# ---------------------------------------------------------------------------
# Run the CLI (via .venv — no uv needed)
# ---------------------------------------------------------------------------

.PHONY: monitor
monitor:
	$(CLI) monitor $(ARGS)

.PHONY: analyze
analyze:
	$(CLI) analyze $(ARGS)

.PHONY: report
report:
	$(CLI) report $(ARGS)

.PHONY: version
version:
	$(CLI) --version

# ---------------------------------------------------------------------------
# Lint & format (via .venv — no uv needed)
# ---------------------------------------------------------------------------

.PHONY: lint
lint:
	$(BLACK) --check $(SRC)
	$(ISORT) --check-only $(SRC)
	$(RUFF) check $(SRC)

.PHONY: format
format:
	$(ISORT) $(SRC)
	$(BLACK) $(SRC)

.PHONY: fix
fix:
	$(RUFF) check --fix $(SRC)
	$(ISORT) $(SRC)
	$(BLACK) $(SRC)

.PHONY: check
check: lint

.PHONY: test
test:
	$(PYTEST) $(TESTS)

# ---------------------------------------------------------------------------
# Clean (system Python — no .venv/uv dependency)
# ---------------------------------------------------------------------------

.PHONY: clean
clean:
	@$(SYSPY) -c "import shutil, pathlib; [shutil.rmtree(p, ignore_errors=True) for p in ['build','dist','.ruff_cache','.pytest_cache','.mypy_cache']]; [shutil.rmtree(p, ignore_errors=True) for p in pathlib.Path('$(SRC)').rglob('__pycache__')]; [shutil.rmtree(p, ignore_errors=True) for p in pathlib.Path('.').glob('*.egg-info')]"

.PHONY: distclean
distclean: clean
	@$(SYSPY) -c "import shutil, pathlib; [shutil.rmtree(p, ignore_errors=True) for p in ['.venv','logs','reports']]; pathlib.Path('uv.lock').unlink(missing_ok=True)"
