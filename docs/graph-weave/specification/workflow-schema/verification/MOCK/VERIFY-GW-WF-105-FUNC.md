# VERIFY-GW-WF-105: List workflows endpoint

> **Linked Task** : GW-WF-105
> **Verification Type** : FUNC
> **Phase ID** : MOCK

---

## Functional Test Cases

1. ✅ GET /workflows?tenant_id=X returns all workflows for tenant
   - Expected: List with count > 0

2. ✅ GET /workflows with status filter returns matching workflows
   - Input: GET /workflows?tenant_id=X&status=active
   - Expected: Only workflows with status=active

3. ✅ GET /workflows with tags filter returns workflows with any matching tag
   - Input: GET /workflows?tenant_id=X&tags=research,earnings
   - Expected: Workflows matching either tag

4. ✅ GET /workflows with owner filter returns workflows by owner
   - Input: GET /workflows?tenant_id=X&owner=research_team
   - Expected: Only workflows with owner=research_team

5. ✅ GET /workflows with no matching filters returns empty list
   - Input: GET /workflows?tenant_id=X&status=archived (when none archived)
   - Expected: 200 OK with empty workflows list, count=0

6. ✅ Combined filters work with AND logic for status/owner, OR for tags
   - Input: GET /workflows?tenant_id=X&status=active&owner=Y&tags=a,b
   - Expected: Workflows where (status=active AND owner=Y AND (tag=a OR tag=b))

7. ✅ Missing tenant_id returns 400 Bad Request

---

## Acceptance Criteria

✅ All 7 functional tests pass
✅ Response includes workflows array and count
✅ Workflows in list response have no definition field (summary only)
✅ Filtering logic matches specification
