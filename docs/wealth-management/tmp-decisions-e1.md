# Decisions made during gap closure

- UNIQ cashback cap is 300,000 VND per month in total (cards.default.json), because WM-RULE-7 states cap 300k and the previous 600k overstated UNIQ cashback.
- ACC-7 balances are changed through transactions only (PATCH /api/accounts edits type and note), because the sheet derives balances from transactions and a written balance would be overwritten or drift.
- Debt-to-income = monthly debt payments / average monthly income x 100 (loan monthly payments plus 5% minimum of each negative account balance), reference 35%, because DTI is a payment-to-income ratio, not a total-debt-to-yearly-income ratio.
- WM-RULE-2 classifies liabilities by balance sign (negative balance = liability), not by account type, because type labels are user text and can be stale while the sign is always correct.
- WM-RULE-9 extra market phases and scenarios already in the engine are kept, because they extend the rule without contradicting it and the UI already renders them.
- WM-RULE-7 billing cycle means the card statement cycle, because cashback caps and utilisation reset per statement, not per rolling 30 days.
- WM-RULE-4 effective dates apply to budgets, cash flow and the new effectiveDate field, but not to daily balance reconstruction (BackCalculateSeries, MonthNetFlow), because money actually moved on the booking date and balances must reconcile with the sheet.
- Error envelope adds code UNAVAILABLE (503) for AI and market services that are not configured or down, because the contract codes only cover the sheet (SHEETS_NOT_CONFIGURED) and reusing that code would tell clients to fix the wrong thing.
- Budget burnOutDate is only set for the current month, because a past month is closed and a future month has no spending pace to extrapolate.
- Budget history tolerates a missing budget tab for earlier years (limits 0) but returns NOT_FOUND when the target month's year has no tab, because older years may predate the sheet while a missing current tab is a setup error.
- Budget reallocation re-reads the budget tab under the write lock and, if the second write fails, restores the first category's original cell, because a half-applied move would silently change the month's total.
- When Google credentials are missing, sheet and analytics routes answer 503 SHEETS_NOT_CONFIGURED without the write guard, because nothing can be read or written and the client should learn which variables to set.
