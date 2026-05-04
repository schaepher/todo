# AGENTS.md — Todo Application

## Project Overview

A standalone local todo system with CLI, Admin API, and server-rendered admin web UI.
Written in Go, backed by SQLite, and designed for single-operator use on one machine.
Implements lifecycle management (open → doing → completed), scheduling, auto-start,
notes, tags, thread links, and full audit history.

## Directory Structure

```
.
├── main.go                 # entry point
├── config/
│   └── config.go           # YAML config loading/saving
├── db/
│   └── db.go               # SQLite connection & migration
│   └── todo.go             # CRUD operations for todos, logs, tags
├── app/
│   └── service.go          # business logic & state transitions
│   ┖── cli.go              # CLI command processing
│   └── api.go              # Admin API HTTP handlers
│   ┖── ui.go               # server–rendered UI handler
│   └── worker.go           # background auto–start worker
│   └── middleware.go       # Bearer token authentication
├── pkg/
│   └── log/
│       └── logger.go       # zap–based logging (GetLogger)
├── templates/
│   ├── layout.html         # page skeleton
│   ├── content.html        # body content (forms, lists)
│   ├── style.html          # CSS
│   ├── script.html         # thin loader for JS class files
│   ├── todo_app.html       # TodoApp: main orchestrator class
│   ├── todo_api.html       # TodoAPI: fetch/API communication class
│   ├── todo_renderer.html  # TodoRenderer: DOM rendering class
│   ├── todo_modal.html     # TodoModal: thread link modal class
│   └── todo_toast.html     # TodoToast: toast notification class
└── go.mod / go.sum         # module & dependencies
```

## Dependencies

- Go 1.21+
- `modernc.org/sqlite` – pure Go SQLite driver
- `gopkg.in/yaml.v3` – YAML config parsing
- `go.uber.org/zap` – structured logging

Install with:

```bash
go mod tidy
```

## Building & Running

### Build binary

```bash
go build -o todo .
```

### Start the Admin server (API + UI + auto-start worker)

```bash
./todo serve
```

### CLI usage

```bash
./todo                     # list open todos
./todo "buy milk"          # add a todo
./todo add "buy milk"      # same as above
./todo list                # list open todos
./todo list all            # list all including completed
./todo detail 1            # show full detail with tags and history
./todo start 1
./todo pause 1
./todo complete 1
./todo complete 1 "done"   # with completion note
./todo reopen 1
./todo schedule 1 "2026-05-05 14:30"
"./todo later 1 "+1h"       # postpone with relative time
./todo note 1 "some note"
./todo delete 1
```

Time formats:

- Absolute: `YYYY-MM-DD HH:MM` (e.g. `2026-05-05 14:30`)
- Relative: prefix `+` followed by duration (e.g. `+10m`, `+1h`, `+1d`)

## Data Directory

All persistent data and config lives under `$TODO_HOME`. When unset, defaults to `$HOME/.todo`. The directory contains:

| File | Purpose |
| --- | --- |
| `todo.db` | SQLite database |
| `config.yaml` | Config file, created automatically on first run |
| `todo.pid` | PID file managed by `make start` / `make stop` |

Config defaults:

```yaml
port: 8080
token: ""                   # empty means no auth required
tag_groups: []              # e.g. - name: priority tags: [high, low]
worker_interval: 30         # seconds between auto-start scans
```

### Tag groups example

```yaml
tag_groups:
  - name: priority
    tags: [urgent, low]
  - name: context
    tags: [home, office]
```

Restart the server after editing the file.

## Logging

Uses `pkg/log` wrapper around `zap`.  
- Production (default): JSON to stderr, info level.  
- Development: `DEBUG=true` switches to text output with colour, debug level.

Inside code, always use `pkglog.GetLogger(ctx)` to obtain the logger. The `ctx` parameter is reserved for future per-request enrichment.

```go
logger := pkglog.GetLogger(r.Context())
logger.Error("message", zap.Error(err))
```

## Code Conventions

- All files are formatted with `gofmt`.
- Error handling: always log errors with the logger, then return the original error.
- Do not swallow errors; the caller decides whether to propagate.
- Global state is kept to a minimum; configuration is loaded once and stored in `config.Get()`.
- SQLite connection (`db.DB`) is initialised once in `main` and closed at shutdown (via `db.DB.Close()`).
- API handlers return JSON with keys `item`, `items`, `message` or an error body.
- UI templates use `{{define "..."}}` and are parsed from `templates/*.html` at init time.
- Auth middleware checks `Authorization: Bearer <token>` header unless config token is empty.
- Token management in the browser uses `localStorage` key `todo_token`.

## Common Development Tasks

### Adding a new CLI command

1. Add a new `case` in `app/cli.go` `RunCLI()`.
2. Call the appropriate service function from `app/service.go` (or add a new one).
3. Print the result or error.

### Adding a new API endpoint

1. Define the endpoint path in `app/api.go` `SetupAPIRoutes`.
2. Add a handler function that parses input, calls service, and returns JSON.
3. Ensure the handler uses `pkglog.GetLogger(r.Context())` for logging.
4. Add matching `case` in the sub-routes of `handleTodoByID` if for a specific todo.

### Adding a new UI feature

1. Edit the relevant template (`content.html` for layout, `todo_*.html` for JS classes).
2. Use `fetch` to call the admin API with the token from `localStorage`.
3. Keep styling aligned with the existing CSS custom properties.
4. For new JS classes, create a new `templates/todo_*.html` with a `{{define "..."}}` block and include it via `{{template}}` in `script.html`.

## Testing

The project currently has no automated tests.  
Manual testing steps:

- Start server and exercise all API endpoints via `curl` or browser.
- Run CLI commands and verify output format.
- Verify scheduling by setting a schedule and waiting for auto-start.
- Verify history by inspecting logs with different lifecycle transitions.

## Architecture Notes

- Same lifecycle service is shared between CLI, API, and auto-start worker.
- View models (`TodoFull`) include tags and logs for the admin UI.
- Delete physically removes the todo and cascades to logs/tags.
- Thread links are optional and editable via modal in the UI.
- `later` is a specific transition (`doing → open`) that requires a target time.
- `reopen` preserves prior completion state in history.

## Delivery Expectations

- The entire system ships as a single binary (with embedded templates via Go’s `embed`).
- All web assets are inlined; no external CDN or separate frontend build pipeline.
- The admin page works on both desktop and narrow screens via responsive two-column layout.
