# Review Remediation TODO (2026-04-22)

## Scope
Based on the latest full-system audit, this file tracks six high-priority remediation items and their implementation plan.

## Priority Summary
- P0-1: Tighten authorization boundaries (owner-or-admin + resource-level checks)
- P0-2: Make upload/version activation transactional and error-safe
- P0-3: Fix batch task status concurrency safety
- P1-4: Align `/programs/export/excel` frontend-backend API contract
- P1-5: Replace lenient ID parsing with strict validation
- P2-6: Introduce server-side pagination/filter pushdown for large datasets

---

## TODO-1 Authorization Hardening (P0)
### Goal
Prevent IDOR and unauthorized data/operation access.

### Implementation
1. Route-level guard:
   - Add owner-or-admin check for `/users/:id`, `/permissions/user/:user_id`, `/permissions/user/:user_id/effective`.
2. Controller-level guard:
   - Enforce resource-level permission for program/file operations by production line capability (`view/download/upload/manage`).
3. Add shared authorization helper middleware/service to avoid duplicated checks.

### Acceptance Criteria
- Non-admin user cannot read/modify other users' data by changing path IDs.
- Program/file operations return 403 when capability is missing.
- Existing admin flow remains functional.

### Risks
- Breaking existing implicit access assumptions in frontend.

---

## TODO-2 Transactional Version Consistency (P0)
### Goal
Ensure upload and version activation remain consistent under failure/concurrency.

### Implementation
1. `UploadFile`:
   - Wrap DB changes in one transaction: file record, version current flag switch, version create/update, program version update.
   - Check all update/save errors.
   - Add failure compensation for file-system write when DB transaction fails.
2. `ActivateVersion`:
   - Use transaction for: clear current + set target current + update program current version.
   - Return explicit failure on any step.
3. Add conflict-safe semantics for concurrent uploads/activations.

### Acceptance Criteria
- No state where a program has zero current version after successful request.
- No silent success when DB update fails.
- Concurrent activation/upload does not produce inconsistent `is_current` state.

### Risks
- Requires careful rollback handling between DB and file system.

---

## TODO-3 Batch Task Concurrency Safety (P0)
### Goal
Eliminate data races in batch import task status updates.

### Implementation
1. Protect all task status reads/writes with mutex or per-task lock.
2. Return immutable snapshot in `GetTaskStatus` instead of shared pointer.
3. Add task lifecycle cleanup policy (TTL cleanup for completed/failed tasks).

### Acceptance Criteria
- `go test -race` (targeted package) shows no race in task status path.
- Status API is stable under frequent polling.
- Completed tasks are cleaned up per TTL policy.

### Risks
- Lock granularity may impact throughput if too coarse.

---

## TODO-4 Export API Contract Alignment (P1)
### Goal
Remove frontend/backend mismatch for `/programs/export/excel`.

### Implementation
Option A (preferred): implement backend route + controller for export.
Option B: remove/replace frontend calls if export is out of scope.

Affected frontend callers:
- `frontend/src/pages/ProgramManagement.tsx`
- `frontend/src/pages/Dashboard.tsx`

### Acceptance Criteria
- No 404 for export action in UI.
- Export behavior documented and consistent across both pages.

### Risks
- Large export may need streaming/async strategy.

---

## TODO-5 Strict ID Parsing and 400 Semantics (P1)
### Goal
Stop silently treating invalid IDs as `0`.

### Implementation
1. Replace `parseUintParam` with strict parser returning `(uint, error)`.
2. For invalid path/query IDs, return `400 Bad Request` with clear message.
3. Update affected handlers in file/program paths.

### Acceptance Criteria
- Invalid IDs return 400, not false 404.
- Error payload is consistent.

### Risks
- Frontend may depend on prior loose behavior; adjust UI error handling if needed.

---

## TODO-6 Server-side Pagination and Filter Pushdown (P2)
### Goal
Improve scalability for large program/user/line/model datasets.

### Implementation
1. Introduce query params: `page`, `page_size`, and filter fields for key list APIs.
2. Backend returns `{items, total, page, page_size}`.
3. Frontend list pages migrate from in-memory full filtering to server-driven pagination.
4. Add indexes for frequent filter/sort fields where needed.

### Acceptance Criteria
- Large dataset pages remain responsive.
- API latency and payload size are bounded.
- Frontend tables functionally match previous behavior.

### Risks
- Requires coordinated contract migration across multiple pages.

---

## Delivery Plan

### Milestone M1 (Week 1)
- TODO-1, TODO-2, TODO-3
- Security and consistency baseline fixed.

### Milestone M2 (Week 2)
- TODO-4, TODO-5
- Contract correctness and error semantics stabilized.

### Milestone M3 (Week 3+)
- TODO-6
- Performance/scalability hardening.

---

## Tracking Checklist
- [ ] TODO-1 Authorization Hardening
- [ ] TODO-2 Transactional Version Consistency
- [ ] TODO-3 Batch Task Concurrency Safety
- [ ] TODO-4 Export API Contract Alignment
- [ ] TODO-5 Strict ID Parsing
- [ ] TODO-6 Server-side Pagination & Filter Pushdown
