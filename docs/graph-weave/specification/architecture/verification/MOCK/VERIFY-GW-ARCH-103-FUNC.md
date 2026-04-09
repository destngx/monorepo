# VERIFY-GW-ARCH-103-FUNC: Mock app core boots

> **Linked Task** : GW-ARCH-103 — `docs/graph-weave/specification/architecture/tasks/MOCK/GW-ARCH-103.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09 00:00
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/architecture/system-architecture.md`
- `docs/graph-weave/specification/architecture/macro-architecture.md`

## 2. Scope Compliance

- ✅ The mock app has a runnable entrypoint and router shell.

## 3. Type-Specific Criteria

| #       | Criterion        | Expected                              | Actual      | Status |
| ------- | ---------------- | ------------------------------------- | ----------- | ------ |
| FUNC-01 | App shell exists | The FastAPI app can start with routes | ✅ Complete | Pass   |

## 4. Evidence

**App Entrypoint**:

- FastAPI app created at `apps/graph-weave/src/main.py`
- Configured with title, description, version
- Swagger UI enabled at `/docs`
- ReDoc enabled at `/redoc`
- OpenAPI schema at `/openapi.json`

**Network Configuration** (NEW):

- App bound to `0.0.0.0:8001` (all network interfaces)
- Accessible from any machine on network: `http://<host-ip>:8001/docs`
- Previously: 127.0.0.1 (localhost only)
- Configuration: `project.json` serve target updated with `--host 0.0.0.0`

**Available Endpoints**:

- ✅ GET /health (health check)
- ✅ GET /docs (Swagger UI)
- ✅ GET /redoc (ReDoc documentation)
- ✅ GET /openapi.json (OpenAPI schema)
- ✅ POST /execute (workflow execution)
- ✅ GET /execute/{run_id}/status (execution status)
- ✅ POST /invalidate (cache invalidation)

**Test Coverage**: All 4 bootstrap tests passing

- ✅ test_app_shell_boots: App can be instantiated
- ✅ test_openapi_docs_available: OpenAPI endpoints respond
- ✅ test_app_has_title: App metadata present
- ✅ test_app_has_version: Version properly configured

## 5. Final Decision

| Decision        | Condition                         |
| --------------- | --------------------------------- |
| Pass            | App core boots successfully       |
| Needs Revision  | App shell is incomplete           |
| Fail + Rollback | Bootstrapping conflicts with spec |

**Decision:** ✅ Pass - App fully bootstrapped with network accessibility
