# Architecture Governance

This project keeps user-visible behavior stable by default. Structural changes should be made in small batches with tests passing after each batch.

## Backend Boundaries

New backend business code should follow:

```text
router -> controller -> service -> repository/db
```

- Router: route grouping and middleware only.
- Controller: parse HTTP input, call services, and shape HTTP responses.
- Service: authorization policy, transactions, business rules, cache invalidation, and cross-table consistency.
- Repository/db: direct persistence details. Existing code may still use `database.DB` directly; new shared persistence logic should move behind service helpers when it affects more than one controller.

## Authorization

Authorization is a backend concern. Frontend menu hiding is only ergonomics, not a security boundary.

- Use `services.IsSystemAdminRole` for `admin/system_admin` checks.
- Use `services.AuthorizeOwnerOrAdmin` for "self or admin" user-resource endpoints.
- Use `services.CheckLineAction` / `services.ResolveAuthorizedLineIDs` for production-line scoped actions.
- Use `services.AuthorizeLineAdminScope` for line-admin management scope.
- Keep controller wrappers thin: read Gin context, call the service policy, return the decision.

## Permission Data

`PermissionRule` is the runtime and business write source for production-line permissions.

- Legacy permission tables may remain for migration compatibility, but new business writes should not target them.
- Permission writes should go through `services.SavePermissionRuleChanges` or `services.SavePermissionRuleChangesTx`.
- Any write path that affects effective permissions must invalidate permission caches after a successful transaction.

## Frontend Boundaries

Frontend refactors should be behavior-frozen unless a feature explicitly requires UX changes.

- Typed API clients live beside the feature, for example `frontend/src/pages/program-management/programApi.ts`.
- Page components compose hooks, API clients, column definitions, and dialogs; they should not accumulate new endpoint strings.
- Feature hooks own loading/data state. Shared auth-derived capabilities should come from `AuthContext` or a feature hook.

## Local Artifacts

Generated or local-only artifacts should stay out of source paths:

- logs, diffs, deployment zips, and performance traces belong in ignored local folders such as `backups/` or `.perf-logs/`.
- disabled historical pages stay ignored under `frontend/src/pages_disabled/` until intentionally restored or deleted in a dedicated cleanup.
- database/operator notes such as `sql.md` are not committed unless explicitly required for a delivery.
