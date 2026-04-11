# GW-MVP-RUNTIME-210: GitHub Token Setup CLI Script

**Objective**: Implement interactive CLI script for GitHub Copilot authentication and credential setup, allowing developers to configure AI provider tokens locally.

**Phase**: [MVP]

## Requirements

### Functional

- Script must support GitHub.com (OAuth device flow) and GitHub Enterprise (token entry) authentication
- Script must support 4 providers: GitHub Copilot, OpenAI, Google Gemini, Anthropic Claude
- GitHub OAuth flow must:
  - Request device code from `https://github.com/login/device/code`
  - Display user code and verification URI
  - Poll `https://github.com/login/oauth/access_token` until authorized or expired
  - Handle expiration, denial, and slow_down errors gracefully
- After successful GitHub auth, script must fetch available models from `https://models.inference.ai.azure.com/models`
- Script must save credentials to `apps/graph-weave/.env.local` (not committed)
- Script must be executable via: `bunx nx run graph-weave:setup-github-token`
- Script must create parent directories if `.env.local` doesn't exist
- Script must handle non-interactive environment gracefully

### Non-Functional

- Script must complete authentication within 15 minutes (GitHub device flow timeout)
- Token must be persisted immediately after auth succeeds
- No credentials logged to stdout (except user code which is temporary)
- Script must support Python 3.8+

## Implementation Approach

1. Create `/apps/graph-weave/scripts/setup_github_token.py` as executable Python script
2. Implement CLI menu using simple prompts (no external CLI frameworks)
3. Implement GitHub OAuth device flow with polling and error handling
4. Add model fetching after token obtained
5. Create helper function `update_env_var()` to safely manage `.env.local`
6. Add Nx target in `project.json` to invoke script with proper venv
7. Create `GITHUB_TOKEN_SETUP.md` documentation with examples

## Acceptance Criteria

- [ ] Script located at `/apps/graph-weave/scripts/setup_github_token.py` and executable
- [ ] Script has interactive provider menu (1-4 selection)
- [ ] GitHub.com flow: requests device code, displays user code + verification URI, polls until auth complete
- [ ] GitHub Enterprise flow: accepts token input, validates, saves
- [ ] Alternative providers: prompts for API key, saves with correct env var name
- [ ] Model fetching: calls Azure endpoint after GitHub auth, displays results
- [ ] Credentials saved to `.env.local` (parent dir created if missing)
- [ ] Nx target `setup-github-token` working: `bunx nx run graph-weave:setup-github-token`
- [ ] Path bug fixed: `.env.local` created in correct location (not doubled path)
- [ ] All error cases handled: expired token, denied access, network timeouts
- [ ] Script syntax validated with Python compiler
- [ ] No credentials in stdout/logs

## Related Requirements

- [[GW-MVP-RUNTIME-202A]] (Real GitHub Provider): Needs GITHUB_TOKEN to fetch real models
- [[GW-MVP-E2E-002]] (Agent + MCP Tools): Validates token works with actual workflow execution

## Deliverables

1. `apps/graph-weave/scripts/setup_github_token.py` (200 lines, executable)
2. `apps/graph-weave/project.json` (updated with setup-github-token target)
3. `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-210-VERIFICATION.md`

## Implementation Notes

- Decision: Script only, not API endpoint (setup is one-time developer flow)
- Decision: Python, not TypeScript (matches GraphWeave stack)
- Decision: Support 4 providers for extensibility (GitHub primary, others future-proof)
- URL scheme fix: Changed from `Path(os.getcwd()) / "apps" / "graph-weave"` to `Path.cwd()` because Nx sets cwd to app directory
- Model fetching endpoint: `https://models.inference.ai.azure.com/models` (Azure Inference API)
- Response format: JSON array with `name`, `friendly_name`, `task` fields

## Environment Variables Set

| Variable                       | Source               | Used By                                      |
| ------------------------------ | -------------------- | -------------------------------------------- |
| `GITHUB_TOKEN`                 | Script (GitHub auth) | Runtime provider factory, skill loading, E2E |
| `OPENAI_API_KEY`               | Script (key entry)   | Runtime provider factory (FULL phase)        |
| `GOOGLE_GENERATIVE_AI_API_KEY` | Script (key entry)   | Runtime provider factory (FULL phase)        |
| `ANTHROPIC_API_KEY`            | Script (key entry)   | Runtime provider factory (FULL phase)        |

## Future Enhancements (FULL Phase)

- Model caching to avoid repeated fetches
- Per-provider token refresh logic
- Azure Entra ID support for production deployments
- Token scope validation before workflow execution
- Per-tenant credential isolation
