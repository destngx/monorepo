---
name: vnstock-migration-expert
description: Expert skill to assist users in migrating code within the Vnstock ecosystem. Handles upgrades from the Free tier to the Sponsor tier (`vnstock_data`), and transitions to the Unified UI (v3.0.0+).
---

# vnstock-migration-expert

> **PURPOSE**: Act as a technical guide and mechanic to safely migrate user code when upgrading to the Sponsor package or adopting the latest Unified UI architecture (v3.0.0+).

## 🚀 QUICK REFERENCE

| User Intent                                | Best Tool / Script                                        | Action                                                                       |
| :----------------------------------------- | :-------------------------------------------------------- | :--------------------------------------------------------------------------- |
| "Add API Key but still community version?" | Guide `docs/vnstock/12-migration-guide.md`                | Explain `vnstock` vs `vnstock_data`. Instruct to install and switch imports. |
| "Migrate large project Free to Sponsor"    | `scripts/migrate_to_vnstock_data.py`                      | Ensure code is committed, then run automated Find & Replace script.          |
| "GUI Installation Failed"                  | `docs/setup-and-debug/02-installation-troubleshooting.md` | Instruct to install from `requirements.txt` first.                           |
| "externally-managed-environment error"     | `docs/setup-and-debug/02-installation-troubleshooting.md` | Instruct to create a Virtual Environment (`.venv`).                          |
| "Migrate old code to Unified UI (v3)"      | `docs/vnstock-data/14-unified-ui.md`                      | Analyze code and map to the 7 Layers (Reference, Market, Fundamental, etc.). |

---

## ⚡ TRIGGER DETECTION

**ACTIVATE WHEN:**

1. User shows logs with `Community version` despite having an active API Key or being a Sponsor.
2. User explicitly requests to "migrate", "convert", or "upgrade" their `vnstock` code.
3. User asks about `show_api()`, Unified UI, Market, Reference, or Fundamental layers.

**DO NOT ACTIVATE WHEN:**

1. The error is unrelated to imports or package installation (e.g., calculation logic errors).
2. The user intentionally uses the free `vnstock` package and does not want to upgrade.

---

## ⚠️ CRITICAL RULES

> [!CRITICAL]
> **1. MANDATORY BACKUP BEFORE MIGRATION**
> Never run `migrate_to_vnstock_data.py` without verifying that the user has committed their code using Git or has a backup. Do not destroy user code.

> [!CRITICAL]
> **2. VIRTUAL ENVIRONMENT IS MANDATORY FOR SPONSOR TIER**
> Before installing `vnstock_data` or using Unified UI, ALWAYS verify the user is inside a `.venv` virtual environment (`source .venv/bin/activate` or `.venv\Scripts\activate`).

> [!IMPORTANT]
> **3. DO NOT BASH-SCRIPT TEXT REPLACEMENT**
> Avoid using `sed` or `awk` for migrations. Use the provided Python script:
> `python .agents/skills/vnstock-migration-expert/scripts/migrate_to_vnstock_data.py <path>`

---

## ⛔ ANTI-PATTERNS

| ❌ AVOID                                                  | ✅ PREFER                                                                                 |
| :-------------------------------------------------------- | :---------------------------------------------------------------------------------------- |
| Running `sed -i` across files.                            | Using the provided `migrate_to_vnstock_data.py` script.                                   |
| Long explanations about API Keys not working.             | Telling them directly: "Change the import package name from `vnstock` to `vnstock_data`". |
| Calling old legacy classes (e.g., `LegacyMacro`) in v3.0. | Mapping strictly to Unified UI (`Macro().economy().gdp()`).                               |

---

## ⚙️ MULTI-STEP WORKFLOW

Follow these exact steps when requested to migrate code.

### Step 1: Context & Backup

- Read the main target files to estimate scope.
- **Exit Condition**: Ask the user: "Is your code committed to Git or backed up?" Proceed only on confirmation.

### Step 2: Automated Refactor

- Execute the migration script in dry-run mode:
  ```bash
  python .agents/skills/vnstock-migration-expert/scripts/migrate_to_vnstock_data.py --dry-run .
  ```
- If the user approves, run without `--dry-run`.
- **Expected Outcome**: All `import vnstock` are replaced with `import vnstock_data`.

### Step 3: Unified UI Adjustments (Optional)

_(Only if migrating to v3.0.0+ Unified UI)_

- Search for legacy instantations. Examples:
  - `Quote(symbol="TCB")` -> `mkt = Market(); mkt.equity("TCB")`
  - `Company(symbol="TCB")` -> `ref = Reference(); ref.company("TCB")`
- Refactor carefully using block replacements, not full file overwrites.

### Step 4: Verification

- Run a basic script (e.g., `Company.overview()`) to verify imports work and API Key is valid.
- Announce migration completion.

---

## 📖 WORKED EXAMPLES

### Example 1: Free to Sponsor Upgrade

**User**: "I bought the Gold package and added the API key but it still says Community!"

**Agent Thinking:**

- _Trigger_: "still says Community" with API key.
- _Action_: Explain `vnstock` vs `vnstock_data` and perform Find & Replace.

**Agent Action:**

1. Reply: "The API key is correct! You simply need to change the library name from `vnstock` to `vnstock_data` in your imports."
2. Propose to change `main.py`:
   ```python
   # Old
   from vnstock import Listing
   # New
   from vnstock_data import Listing
   ```

### Example 2: Legacy to Unified UI Migration

**❌ LEGACY CODE (Pre v3.0):**

```python
from vnstock_data import Finance, Quote
fin = Finance(symbol="TCB")
print(fin.ratio())

price = Quote(symbol="TCB").history(start="2026-01-01", end="2026-03-01")
```

**✅ MIGRATED CODE (Unified UI v3):**

```python
from vnstock_data import Fundamental, Market
fun = Fundamental()
print(fun.equity("TCB").ratio())

mkt = Market()
price = mkt.equity("TCB").ohlcv(start="2026-01-01", end="2026-03-01")
```

---

## 📋 QUALITY CHECKLIST

Before notifying the user of completion, verify:

- [ ] Code was backed up prior to migration.
- [ ] No residual `import vnstock` remains (if upgrading to sponsor tier).
- [ ] Unified UI correctly utilizes the 7 Layers (Reference, Market, Fundamental, etc.).
- [ ] No hallucinated API endpoints were used (checked via `show_api()`).
- [ ] Linked `12-migration-guide.md` or `14-unified-ui.md` for further reading.
