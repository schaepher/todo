# Todo Module Requirements Spec

## 1. Scope

- Todo module is a standalone local workflow system for individual operators.
- It must provide three supported surfaces:
  - command-line interface
  - Admin API
  - Admin web UI
- Chat or agent entrypoints are out of scope.
- The system is local-first rather than cloud-synchronized.

## 2. System Principles

- CLI, Admin API, and Admin UI must share one todo lifecycle model.
- The Admin UI is a first-class operating console rather than a demo page.
- The UI should stay lightweight and rely on a backend-rendered page plus small amounts of native JavaScript.
- Todo history must remain auditable across normal lifecycle operations.

## 3. Persistence and Data Model

- Todo data is persisted locally in SQLite at `$TODO_HOME/todo.db`.
- Configuration is stored at `$TODO_HOME/config.yaml`.
- `TODO_HOME` defaults to `$HOME/.todo` and is overridable via the `TODO_HOME` environment variable.
- A todo item must support at least:
  - `id`
  - `content`
  - `status`
  - `created_at`
  - `thread_link`
  - `scheduled_at`
  - `completed_at`
- Todo history must be stored separately from the main todo row.
- Tags must be stored separately from the main todo row so one todo can carry multiple tags.
- Tag groups are configured in the system configuration file.
- Tag-group configuration must support:
  - group name
  - candidate tag list

## 4. State Model

- Supported states are:
  - `open`
  - `doing`
  - `completed`
- `open` means not yet started or moved back from active work.
- `doing` means actively in progress.
- `completed` means logically finished.
- `complete` is not the same as `delete`.
- `delete` is physical removal.

Allowed lifecycle transitions:

- `open -> doing`
- `doing -> open`
- `doing -> open` via `later`
- `open -> completed`
- `doing -> completed`
- `completed -> open`

State rules:

- `scheduled_at` may be set only while a todo is `open`.
- `later` may be used only while a todo is `doing`.
- `start` clears any saved schedule.
- `complete` clears any saved schedule.

## 5. Core Behavior Requirements

- CLI add-or-list behavior must work as follows:
  - non-empty content adds a todo
  - empty or whitespace-only content lists active todos
- Default list behavior is open-plus-doing only; completed items are hidden unless explicitly requested.
- Todo notes and completion notes are first-class behaviors.
- Reopen must preserve prior history rather than creating a fresh unrelated item.
- Delete must remove the todo and its dependent records.

## 6. Main Link Requirements

- A todo may carry a `thread_link` that points to the related main discussion.
- `thread_link` may be provided at creation time.
- `thread_link` may be updated later.
- `thread_link` may be cleared by saving an empty value.
- The UI must always render a main link row even when the value is empty.
- No URL scheme validation is performed; any scheme (`xxx://`) is accepted.

## 7. Tag Requirements

- Tag groups must be rendered from configuration rather than hard-coded in the page.
- A todo may carry multiple tags from the same group.
- Tag add must be idempotent for the same `(group_name, tag)` pair.
- Existing tags must be removable directly from the todo item.
- Search Todo List must support filtering by selected tags.
- Tag-filter semantics must match current behavior:
  - within the same group, multiple selected tags behave as OR
  - across different groups, selected tags behave as AND

## 8. Schedule and Later Requirements

- Any `open` todo may set a schedule time.
- Any `open` todo may update that schedule time.
- Any `open` todo may clear the schedule by saving an empty value.
- When current time reaches `scheduled_at`, the system automatically starts that todo.
- Auto-start must reuse the normal start semantics rather than introducing a separate status path.
- `later` is part of the schedule family and applies only to `doing` todos.
- `later` means:
  - move the todo from `doing` back to `open`
  - set `scheduled_at`
  - let auto-start move it back to `doing` later
- `later` requires a target time and must not succeed without one.
- Schedule-related UI must provide quick offset buttons:
  - `+10m`
  - `+30m`
  - `+1h`
  - `+1d`

## 9. CLI Requirements

- CLI must support at least:
  - list
  - add
  - detail
  - note
  - start
  - pause
  - schedule
  - later
  - complete
  - reopen
  - delete
- CLI output should remain stable and ID-forward so follow-up operations are easy to run.
- CLI validation errors should explain invalid state in operator language.

## 10. Admin API Requirements

- The Admin surface must expose both page rendering and JSON mutation APIs.
- The todo API surface must include:
  - `GET /admin/api/todos`
  - `POST /admin/api/todos`
  - `POST /admin/api/todos/{id}/note`
  - `DELETE /admin/api/todos/{id}/notes/{log_id}`
  - `POST /admin/api/todos/{id}/thread`
  - `POST /admin/api/todos/{id}/tags`
  - `DELETE /admin/api/todos/{id}/tags/{tag_id}`
  - `POST /admin/api/todos/{id}/schedule`
  - `POST /admin/api/todos/{id}/later`
  - `POST /admin/api/todos/{id}/start`
  - `POST /admin/api/todos/{id}/pause`
  - `POST /admin/api/todos/{id}/reopen`
  - `POST /admin/api/todos/{id}/complete`
  - `DELETE /admin/api/todos/{id}`
- `GET /admin/api/todos` must support optional inclusion of completed items.
- Mutation APIs should return refreshed item data when the UI needs immediate re-render after success.

## 11. Admin UI Requirements

### 11.1 Page Structure

- The Todo page contains the following major sections in order:
  - page header
  - `Doing List`
  - `Todo List`
  - `Last Result` — collapsible, hidden by default
- `Doing List` must appear above `Todo List`.
- `Last Result` is a collapsible section that shows the latest todo-action response on demand.
- English characters must be rendered in a monospace font.

### 11.2 New Todo Modal

- Adding a todo opens a modal overlay rather than an inline form.
- The modal must contain a single form with these fields:
  - `Content` multiline textarea
  - `Main Link` text input (trimmed, no scheme validation)
  - `Auto Start At` datetime-local input with quick offset buttons (`+10m`, `+30m`, `+1h`, `+1d`)
- If tag groups are configured, the modal must render tag checkboxes by group for selection at creation time.
- The form must expose an `Add Todo` submit button.
- On success, the modal closes, the form resets, and the list refreshes.
- A `Cancel` button closes the modal without changes.

### 11.3 Doing List

- Doing items must be rendered in a dedicated list.
- Doing items must be fully expanded by default.
- Doing items remain in this list until paused, moved to later, or completed.
- Empty state text must communicate that the doing list is empty.

### 11.4 Todo List Header and Toggles

- The `Todo List` title must include remains count in the form `Todo List (remains: N)`.
- The header must expose a `New Todo` button that opens the New Todo modal.
- The count line (`Showing N item(s).`) must contain the two toggles inline:
  - `Show completed todos`
  - `Show details`
- `Show completed todos` controls whether completed items are included from backend data.
- `Show details` expands all non-doing cards.
- Doing items must remain expanded even when `Show details` is off.

### 11.5 Search and Tag Filter Area

- The search area must appear above the main `Todo List`.
- It must contain:
  - `Search Todo List` search input
  - hint text clarifying that search only filters `Todo List` below
  - a collapsible tag-filter panel with a `Filter By Tag` toggle button (default collapsed)
- The tag-filter panel's collapsed/expanded state must be remembered across re-renders in the current session.
- Search must match the same fields as the current implementation:
  - todo ID
  - status
  - content
  - main link
  - tag group names
  - tag values
  - note text
  - note kind
  - note timestamp text
  - completion-note text
  - completion-note kind
  - completion-note timestamp text
- The tag-filter panel must appear only when configured tag groups exist.
- Each tag group must be rendered on its own line within the panel.
- When no tag filter is active, the panel must explain that configured tags can narrow the list.
- When tag filters are active, the panel must show:
  - active tag-filter count
  - explanation that same-group selections are OR
  - `Clear filters` button

### 11.6 Todo List Counts and Empty States

- The UI must display a count line in the form `Showing N item(s).`
- This visible-item count must reflect the filtered `Todo List`, not the total todo dataset.
- The `Show completed todos` and `Show details` toggles must appear inline on the same line as the count.
- Empty-state messages must distinguish:
  - no items in `Doing List`
  - no items in `Todo List`
  - no items matched current filters

### 11.7 Todo Card Summary

- Each todo card summary must show:
  - `#id`
  - `Status: ...`
  - `Created: ...`
  - `Auto Start: ...` when `scheduled_at` exists
  - `Completed: ...` when `completed_at` exists
  - main content in full (no truncation or line clamp)
- If tags exist, the summary must also show a tag row with selected tags only (no inline tag editors).
- The summary must always show a main link row with:
  - `Main Link:` label
  - current link or empty-state copy
  - `Edit` button

Summary actions:

- `open` item:
  - `Start`
  - `Complete`
  - optional `Details`
  - `Delete`
- `doing` item:
  - `Pause`
  - `Delete`
- `completed` item:
  - `Reopen`
  - optional `Details`
  - `Delete`

- `Details` action is hidden for doing items because they are always expanded.
- The quick `Complete` action is no longer available on doing items; completion is handled from the details panel.

### 11.8 Todo Card Details

- The details area must render below the summary.
- It must contain, in order:
  - `Tags` section
  - `Notes` section
  - `Completion Notes` section (when completion notes exist)
  - `Add Note` form (shared with Complete)
  - schedule or later form depending on state

Tags section requirements:

- Existing tags must render as `[group: tag]` chips in a single row (informational display only, no remove control on chips).
- If no tags exist, the section must show an empty-state hint (`No tags.`).
- An `Edit Tags` button opens the Edit Tags modal when tag groups are configured.
- Tag editing is done entirely through the Edit Tags modal, not inline in the details panel.

Edit Tags modal requirements:

- The modal must display a `Selected Tags` area showing currently assigned tags (informational, no remove button).
- The modal must display `Available Tags` organized by group with clickable tag buttons.
- Clicking an unselected available tag adds it to the todo immediately via API.
- Clicking a selected available tag removes it from the todo immediately via API.
- No explicit save button is needed; each toggle calls the API directly.
- The modal content refreshes after each toggle to reflect the updated state.

Notes section requirements:

- Regular notes must show:
  - timestamp text
  - kind text
  - rendered note body
- Regular notes must expose a `Delete` button.
- Notes must be displayed in chronological order (oldest first).
- Completion notes must show the same metadata and content shape.
- Completion notes must not expose a delete button.
- Empty note lists must show `None`.

### 11.9 Note and Completion Forms

- `Add Note` form must expose:
  - multiline textarea (shared with Complete)
  - `Save Note` button
  - `Complete Todo` button
- `Complete Todo` uses the content of the shared textarea as the completion note.
- There is no separate `Completion Note` form.
- Summary-level `Complete` on open items must still exist as a quick action with an empty note.

### 11.10 Schedule and Later Forms

- `open` items must expose an `Auto Start At` form in details.
- This form must contain:
  - datetime-local input (narrow width, `14rem`)
  - submit button whose label changes between save and update based on current value
  - quick offset buttons `+10m`, `+30m`, `+1h`, `+1d` placed inline next to the datetime input
  - hint that empty value clears the schedule

- `doing` items must expose a `Later Until` form in details.
- This form must contain:
  - datetime-local input (narrow width, `14rem`)
  - `Later` submit button
  - quick offset buttons `+10m`, `+30m`, `+1h`, `+1d` placed inline next to the datetime input
  - hint that later moves the todo back to open and auto-starts it at the selected time

- `completed` items must expose neither schedule nor later forms.

### 11.11 Main Link Modal

- Clicking the summary `Edit` button opens a dedicated main-link modal.
- The modal must contain:
  - dynamic title including todo ID
  - hidden todo ID field
  - `Main Link` text input
  - hint that empty value clears the link
  - `Cancel` button
  - `Save Main Link` button
  - top-level `Close` button
- On open, the input must receive focus and its content should be selected.
- The modal must close when:
  - clicking backdrop
  - pressing `Escape`
  - clicking `Close`
  - clicking `Cancel`
  - successful save

### 11.12 Feedback Behavior

- Every successful todo mutation should:
  - refresh todo data
  - update `Todo Result`
  - show a success toast
- Failed todo mutations should show an error toast.

## 12. Security and Validation Requirements

- Admin access must be authenticated.
- Todo content, notes, and main links must be treated as untrusted input.
- Main links must be clearable.
- Delete behavior must remain explicit and auditable.

## 13. Non-Functional Requirements

- Todo history must remain auditable after note deletion, completion, reopen, schedule update, and later.
- Auto-start behavior must be observable through explicit success and failure signals.
- The system is optimized for local operator-sized datasets.
- UI filtering should feel immediate for local datasets.

## 14. Service Management

- The service provides a Makefile with `start`, `stop`, and `restart` targets for process lifecycle management.
- On `start`, a PID file is written to `$TODO_HOME/todo.pid`.
- On `stop`, the PID file is removed.
- If the PID file already exists and the process is running, `start` skips without starting a duplicate.
- The database file is always at `$TODO_HOME/todo.db` — the path is derived from `TODO_HOME`, not configurable.
