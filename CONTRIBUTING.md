# Contributing to Sift MCP

Thanks for your interest in contributing to **Sift MCP**! This project is an MCP server for test intelligence (ingesting test reports, extracting root causes, fingerprinting, history, trends).

## Quick links

- Repo: https://github.com/sift-mcp/sift-mcp
- Issues: https://github.com/sift-mcp/sift-mcp/issues

If you're looking for a good place to start, check issues labeled **good first issue** or **help wanted**.

## Code of conduct

Be respectful, assume good intent, and keep discussions constructive.

## Licensing

- Open-source license: **AGPL-3.0** (see `LICENSE`)
- Commercial license options may be available (see `COMMERCIAL_LICENSE.md`)

By contributing, you agree your contributions are licensed under the project’s open-source license unless otherwise agreed.

## Development setup

### Prerequisites

- Go **1.24+**

### Clone

```bash
git clone https://github.com/sift-mcp/sift-mcp.git
cd sift-mcp
```

### Build

```bash
go build -o sift ./cmd/server
```

### Run locally

Sift runs as an MCP server over stdio.

```bash
SIFT_DB_PATH=./sift.db ./sift
```

### Run tests

```bash
go test ./...
```

## Project structure (high level)

A few helpful entry points:

- `cmd/server/main.go` — binary entrypoint
- `internal/server/factory.go` — wires up DB, parser, analysis pipeline, MCP server
- `internal/mcp/` — MCP protocol handling + tool definitions (see `internal/mcp/tools.go`)
- `internal/parser/` — report parsers (currently `junit_parser.go`)
- `internal/analysis/` — analysis pipeline + stages
- `internal/database/` — SQLite provider, schema, repositories

## How to contribute

### 1) Pick an issue

- Comment on the issue to claim it.
- If you’re proposing a larger change, open an issue first so we can align on approach.

### 2) Create a branch

Use a descriptive name, e.g.

- `feat/parser-registry`
- `fix/junit-missing-classname`
- `test/add-junit-fixtures`

### 3) Make focused commits

Small, reviewable commits are preferred.

### 4) Open a PR

Your PR description should include:

- What changed and why
- How it was tested (`go test ./...`)
- Any screenshots/logs if relevant

## Adding or improving a parser (most wanted)

Sift currently ingests JUnit XML but is designed to expand to additional formats.

### Where ingestion happens

- MCP tool entrypoint: `internal/mcp/tools.go` (`ingest_report`)
- Parser implementation: `internal/parser/junit_parser.go`
- Wiring: `internal/server/factory.go` (`CreateParser()` and `NewTools(...)`)

### Recommended approach for new formats

1. Add a new parser under `internal/parser/` (e.g. `trx_parser.go` or `nunit_parser.go`).
2. Keep it streaming-friendly when possible (avoid loading huge files unnecessarily).
3. Convert the input format into the internal domain model (`core.TestReport`, suites, test cases, failures).
4. Add unit tests and fixtures.

> Note: there is an open issue to introduce a parser registry and format selection so multiple parsers can coexist cleanly.

## Fixtures and golden tests

For parser work, we strongly prefer adding real-world fixtures.

Suggested convention:

- Put sample reports under `testdata/` (e.g. `testdata/junit/pytest.xml`)
- Add parser tests that:
  - parse fixture
  - assert totals (tests/failed/skipped)
  - assert extracted failures (names/messages)
  - verify edge cases (missing attributes, nested suites)

## Style guidelines

- Prefer clarity over cleverness.
- Keep functions small and testable.
- Avoid adding heavy dependencies unless there’s a strong justification.

## Reporting security/privacy concerns

If you find a security issue, please open a GitHub Security Advisory or contact the maintainers privately rather than posting sensitive details in a public issue.
