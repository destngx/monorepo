# VERIFY-GW-WF-106: Update workflow endpoint

> **Linked Task** : GW-WF-106
> **Verification Type** : FUNC
> **Phase ID** : MOCK

---

## Functional Test Cases

1. ✅ PUT /workflows/{workflow_id} with valid update returns 200 OK
   - Input: Update name and tags
   - Expected: 200 with updated metadata

2. ✅ Update workflow status from active to archived
   - Input: PUT with status=archived
   - Expected: 200, status changed

3. ✅ Update definition with valid workflow JSON succeeds
   - Input: PUT with new definition
   - Expected: 200, definition updated

4. ✅ Workflow not found returns 404 Not Found
   - Input: PUT /workflows/nonexistent:v1.0.0
   - Expected: 404

5. ✅ Invalid status value returns 422 Validation error
   - Input: PUT with status=invalid_status
   - Expected: 422

6. ✅ Attempt to update version returns 400 (immutable field)
   - Input: PUT with version changed
   - Expected: 400 or rejected silently

7. ✅ Empty request body returns 400
   - Input: PUT with no fields
   - Expected: 400

8. ✅ updated_at timestamp changes, created_at unchanged
   - Setup: Create workflow, note created_at
   - Action: PUT to update
   - Expected: updated_at changed, created_at same

---

## Acceptance Criteria

✅ All 8 functional tests pass
✅ Immutable fields protected
✅ Error responses follow standard format
