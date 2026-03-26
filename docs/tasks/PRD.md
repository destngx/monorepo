# PRD: Wealth Management Code Refactoring (File Splitting)

## 🎯 Objective

Refactor the `wealth-management` codebase by splitting files that exceed 300 lines of code. This will improve code maintainability, readability, and follow the project's architectural standards of keeping components and services focused and modular.

## 📝 User Requirements

1.  **Identify all files** in `libs/wealth-management` and `apps/wealth-management` that are > 300 lines.
2.  **Split components** into smaller, reusable pieces.
3.  **Modularize services** by extracting logic into specialized utilities or sub-services.
4.  **Extract hooks** from large components to separate state and side-effect logic.
5.  **Ensure no regression** in functionality after splitting.
6.  **Maintain consistent styling** and premium aesthetics.

## 🛠 Actionable Tasks

### Phase 1: High-Priority Large Files (> 1000 lines)

- [x] Refactor `apps/wealth-management/src/components/dashboard/market-pulse-dashboard.tsx` (Modularized)
  - [x] Extract `MarketSection` to handle regional market views.
  - [x] Use existing sub-components more effectively.
- [x] Refactor `libs/wealth-management/src/shared/api/market-data-service.ts` (1267 lines)
  - [x] Extract technical analysis logic to `TechnicalAnalysisService.ts`.
  - [x] Extract valuation logic to `ValuationService.ts`.
  - [x] Extract regional pulse logic to `MarketPulseVnService.ts` and `MarketPulseUsService.ts`.
- [x] Refactor `apps/wealth-management/src/features/investments/ui/fmarket-dashboard.tsx` (1046 lines)
  - [x] Extract `FundTable` to a separate component.
  - [x] Extract `GoldProductTable` to a separate component.
  - [x] Extract `TickerDetails` to a separate component.
  - [x] Extract `BankRatesSection` and `GoldUsdSection`.

### Phase 2: Medium-Priority Files (500 - 1000 lines)

- [x] Refactor `libs/wealth-management/src/features/investments/ui/page.tsx` (Modularized)
  - [x] Extract `ThinkTank` and `AssetLedgers`.
- [x] Refactor `apps/wealth-management/src/features/investments/ui/page.tsx` (Modularized)
  - [x] Extract `ThinkTank` and `AssetLedgers`.
- [x] Refactor `apps/wealth-management/src/components/dashboard/ticker-analysis-dashboard.tsx` (604 lines)
  - [x] Extract `TickerValuationView`.

### ✅ Phase 3: Secondary Components & Cleanup (Completed)

- [x] Refactor `apps/wealth-management/src/features/accounts/ui/page.tsx` (483 lines)
  - [x] Extract `AccountSection`.
- [x] Refactor `libs/wealth-management/src/config/transactions/categories.ts` (481 lines)
- [x] Refactor `libs/wealth-management/src/features/accounts/ui/page.tsx` (457 lines)
- [x] Refactor `apps/wealth-management/src/components/transactions/notification-processor.tsx` (431 lines)
- [x] Refactor `apps/wealth-management/src/components/transactions/transaction-form.tsx` (395 lines)
- [x] Refactor `apps/wealth-management/src/features/chat/ui/ai-drawer.tsx` (323 lines)
- [x] Refactor `apps/wealth-management/src/features/chat/ui/chat-interface.tsx` (303 lines)
- [ ] ... and other files identified in the initial scan.

## 📅 Timeline

- Start date: 2026-03-26
- Target completion: 2026-03-28
- Status: Complete
