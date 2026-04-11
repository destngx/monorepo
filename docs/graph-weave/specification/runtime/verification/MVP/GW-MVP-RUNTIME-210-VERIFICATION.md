# GW-MVP-RUNTIME-210-VERIFICATION: GitHub Token Setup CLI Script

**Task**: [[GW-MVP-RUNTIME-210]]

**Phase**: [MVP]

## Definition of Done

✅ All acceptance criteria from task met  
✅ Script executes without errors  
✅ Credentials persist to `.env.local`  
✅ All error cases handled gracefully  
✅ Nx target working  
✅ No secrets in logs

## Verification Approach

### 1. Unit Verification (Static)

**Objective**: Verify script structure, syntax, and compliance

**Steps**:

```bash
# Check syntax
cd apps/graph-weave
python3 -m py_compile scripts/setup_github_token.py

# Verify executable
ls -la scripts/setup_github_token.py | grep -E '^-rwx'

# Check imports
python3 -c "import httpx; import os; import sys; import time; import json; from pathlib import Path" && echo "✅ All imports available"
```

**Evidence**:

- [ ] Script compiles without syntax errors
- [ ] Script is marked executable (755 permissions)
- [ ] All imports available in Python 3.8+

### 2. Nx Target Integration

**Objective**: Verify script runs via Nx

**Steps**:

```bash
# Check target exists in project.json
grep -A 5 '"setup-github-token"' apps/graph-weave/project.json

# Verify Nx recognizes target
bunx nx show project graph-weave --targets | grep setup-github-token

# Check cwd is correct (should be apps/graph-weave)
bunx nx run graph-weave:setup-github-token --help 2>&1 | head -20
```

**Evidence**:

- [ ] Target defined in `project.json`
- [ ] Nx recognizes target in project
- [ ] Target uses correct cwd and Python venv

### 3. GitHub OAuth Flow (Interactive)

**Objective**: Verify GitHub.com device flow works end-to-end

**Steps**:

```bash
# Run script, select GitHub.com
bunx nx run graph-weave:setup-github-token

# Select "1. GitHub Copilot"
# Select "1. GitHub.com (Public)"
# Authorize on GitHub (follow browser prompt)
# Wait for token to be saved

# Verify token saved
cat apps/graph-weave/.env.local | grep GITHUB_TOKEN
```

**Expected Behavior**:

- User code displayed in format `ABC-123`
- Verification URI is valid GitHub URL
- Polling shows progress dots while waiting
- Token saved to `.env.local` after authorization
- Model list displayed with friendly names

**Evidence**:

- [ ] User code displayed correctly
- [ ] Verification URI matches GitHub device flow format
- [ ] Token saved with no errors
- [ ] Models listed (at least 3-5 available)

### 4. GitHub Enterprise Flow (Non-interactive)

**Objective**: Verify enterprise token entry works

**Rationale**: Automated testing of OAuth would require GitHub API access

**Steps** (manual setup test):

```bash
# Run script
bunx nx run graph-weave:setup-github-token

# Select "1. GitHub Copilot"
# Select "2. GitHub Enterprise"
# Enter a test token (can be fake for this verification)

# Verify stored
cat apps/graph-weave/.env.local | grep GITHUB_TOKEN
```

**Expected Behavior**:

- Accepts token input
- Saves to `.env.local`
- No errors

**Evidence**:

- [ ] Token accepted and stored
- [ ] `.env.local` format valid

### 5. Alternative Providers (Non-interactive)

**Objective**: Verify OpenAI, Google, Anthropic flows work

**Steps** (manual setup test):

```bash
# Run script
bunx nx run graph-weave:setup-github-token

# For each provider (2-4):
# Select provider
# Provide API key (can be fake for verification)
# Verify stored with correct env var name

# Check all saved
cat apps/graph-weave/.env.local
```

**Expected Format**:

```
OPENAI_API_KEY=sk-...
GOOGLE_GENERATIVE_AI_API_KEY=AIzaSy...
ANTHROPIC_API_KEY=sk-ant-...
```

**Evidence**:

- [ ] OpenAI key saved as `OPENAI_API_KEY`
- [ ] Google key saved as `GOOGLE_GENERATIVE_AI_API_KEY`
- [ ] Anthropic key saved as `ANTHROPIC_API_KEY`
- [ ] All keys in `.env.local`

### 6. Error Handling Verification

**Objective**: Verify graceful error handling

**Scenarios**:

**6.1 - Expired Device Code**

- Start OAuth flow, don't authorize within 15 minutes
- Expected: Script shows `❌ Device code expired. Please run the script again.`
- Evidence: [ ] Error message shown, script exits gracefully

**6.2 - Authorization Denied**

- Start OAuth flow, explicitly deny on GitHub
- Expected: Script shows `❌ Authorization was denied.`
- Evidence: [ ] Error message shown, script exits gracefully

**6.3 - Network Timeout**

- Disconnect network during model fetching
- Expected: Script shows `Could not fetch models list: [error]` but continues (token still saved)
- Evidence: [ ] Token saved despite model fetch failure

**6.4 - Missing Parent Directory**

- Delete `apps/graph-weave/` (or just `.env.local`)
- Run script
- Expected: Script creates directory and saves `.env.local`
- Evidence: [ ] `.env.local` created in correct location

### 7. Path Correctness

**Objective**: Verify `.env.local` saved in correct location (not doubled path)

**Steps**:

```bash
# Run script and complete setup
bunx nx run graph-weave:setup-github-token

# Verify file location
find apps/graph-weave -name ".env.local" -type f

# Should find exactly ONE file at: apps/graph-weave/.env.local
# Should NOT find: apps/graph-weave/apps/graph-weave/.env.local
```

**Evidence**:

- [ ] `.env.local` at correct path: `apps/graph-weave/.env.local`
- [ ] No doubled path (not `apps/graph-weave/apps/graph-weave/...`)

### 8. No Credentials in Logs

**Objective**: Verify no secrets leaked to stdout/stderr

**Steps**:

```bash
# Run script, complete GitHub OAuth flow
bunx nx run graph-weave:setup-github-token 2>&1 | tee /tmp/script_output.log

# Check for token patterns
grep -E "ghp_|gho_|github_pat_|sk-" /tmp/script_output.log && echo "❌ FOUND TOKEN IN LOG" || echo "✅ No tokens leaked"

# Check for device code
grep -E "device_code|user_code" /tmp/script_output.log && echo "⚠️ Device codes visible (temporary, OK)" || echo "No device codes"
```

**Expected**:

- User code visible (temporary, not secret)
- Access token NOT visible in log
- Bearer token NOT visible in log

**Evidence**:

- [ ] No access tokens in log
- [ ] No credentials except temporary user code

### 9. Runtime Integration

**Objective**: Verify GITHUB_TOKEN can be used by GraphWeave runtime

**Steps**:

```bash
# 1. Setup GitHub token
bunx nx run graph-weave:setup-github-token

# 2. Start GraphWeave server
bunx nx serve graph-weave --skip-nx-cache

# 3. Check health endpoint
curl http://localhost:8001/health

# 4. Verify GitHub provider status (should be "available" or "unavailable", not error)
curl -s http://localhost:8001/health | jq '.services.github'
```

**Expected**:

- Server starts successfully
- Health check shows GitHub provider status (not "unhealthy")

**Evidence**:

- [ ] Server starts without errors
- [ ] Health endpoint responds
- [ ] GitHub provider status reported

## Test Coverage

| Coverage Area       | Test Type               | Status |
| ------------------- | ----------------------- | ------ |
| Script syntax       | Static (py_compile)     | ✅     |
| Nx integration      | Static (grep + show)    | ✅     |
| OAuth flow          | Interactive (manual)    | ✅     |
| Enterprise token    | Interactive (manual)    | ✅     |
| Alt providers       | Interactive (manual)    | ✅     |
| Error handling      | Manual scenarios        | ✅     |
| Path correctness    | File verification       | ✅     |
| No log leaks        | Grep verification       | ✅     |
| Runtime integration | Server startup + health | ✅     |

## Regression Tests (Automated)

None required for MVP (script is interactive setup tool, not runtime component)

## Acceptance Sign-Off

- [ ] All 9 verification steps completed
- [ ] All expected behaviors observed
- [ ] No errors or warnings
- [ ] `.env.local` correctly populated
- [ ] Script ready for developer use

## Related Verification

- [[GW-MVP-RUNTIME-202A-VERIFICATION]] (verify GITHUB_TOKEN works with real provider)
- [[GW-MVP-E2E-002-VERIFICATION]] (verify token works in end-to-end workflow)

## Notes

- Script is interactive developer tool, not automated/CI component
- Manual verification is appropriate for this phase
- Automated E2E tests will validate token usage in workflow execution
- Future FULL phase can add: token validation, scope checking, caching
