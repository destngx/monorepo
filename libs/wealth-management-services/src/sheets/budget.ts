import { readSheet } from './client';
import { BudgetItem } from '../types/budget';
import { getCached, setCache } from '../db/cache';

const CACHE_KEY = 'budget';

// The Budget_2026 sheet layout:
// Row 10:  A="", B="", C=yearly serial, D=Jan serial, E=Feb serial, ... (col index 2=yearly, 3=month1 ...)
// Rows 12+: Income categories  A=category, B="", C=yearly limit, D=jan limit, E=feb ...
// Row 43:  "Total Income" separator
// Rows 45+: Expense categories (same layout)
// Category rows that are section headers (ALL_CAPS, no data) are skipped.

const SECTION_HEADERS = new Set(['help', 'income categories', 'expense categories', 'transfer categories']);
const JUNK_PREFIXES = ['total ', 'net ', 'projected', '[', 'starting balance'];

function isJunk(name: string): boolean {
  if (!name?.trim()) return true;
  const lower = name.trim().toLowerCase();
  if (SECTION_HEADERS.has(lower)) return true;
  if (JUNK_PREFIXES.some(p => lower.startsWith(p))) return true;
  // Skip pure numeric or empty-ish cells
  if (/^\d+$/.test(lower)) return true;
  return false;
}

function num(v: string | undefined): number {
  if (!v && v !== '0') return 0;
  const n = parseFloat(String(v).replace(/,/g, ''));
  return isNaN(n) ? 0 : n;
}

// Convert a Google Sheets serial date number to { month (0-based), year }
function serialToMonthYear(serial: number): { month: number; year: number } {
  const epoch = new Date(1899, 11, 30);
  const d = new Date(epoch);
  d.setDate(d.getDate() + Math.round(serial));
  return { month: d.getMonth(), year: d.getFullYear() };
}

export async function getBudget(forceFresh = false): Promise<BudgetItem[]> {
  if (!forceFresh) {
    const cached = await getCached<BudgetItem[]>(CACHE_KEY);
    if (cached) return cached;
  }

  // Read sheet wide enough to capture all 12 months (cols A-N = columns 0-13)
  const rows = await readSheet('Budget_2026!A1:N200');

  // Row 10 (index 9) contains the month serial numbers starting at col C (index 2)
  const headerRow = rows[9] ?? [];
  // Build a map from column index → { month, year }
  // col index 2 = "Yearly" (skip), col index 3 = first month, etc.
  const colToMonth: Record<number, { month: number; year: number }> = {};
  for (let c = 3; c < headerRow.length; c++) {
    const serial = num(headerRow[c]);
    if (serial > 1000) {
      colToMonth[c] = serialToMonthYear(serial);
    }
  }

  const items: BudgetItem[] = [];
  let currentSection: 'income' | 'expense' | 'non-budget' = 'non-budget';

  for (let r = 11; r < rows.length; r++) {
    const row = rows[r];
    const category = row[0]?.trim() ?? '';
    if (!category) continue;

    const lower = category.toLowerCase();
    if (lower === 'income categories') {
      currentSection = 'income';
      continue;
    }
    if (lower === 'expense categories') {
      currentSection = 'expense';
      continue;
    }
    if (lower === 'transfer categories') {
      currentSection = 'non-budget';
      continue;
    }

    if (isJunk(category)) continue;

    const yearlyLimit = num(row[2]);

    // Sum monthly limits from each month column
    let monthlyLimitSum = 0;
    const monthlyLimits: Record<string, number> = {};
    for (const [colStr, mth] of Object.entries(colToMonth)) {
      const col = Number(colStr);
      const v = num(row[col]);
      if (v !== 0) {
        const key = `${mth.year}-${String(mth.month + 1).padStart(2, '0')}`;
        monthlyLimits[key] = v;
        monthlyLimitSum += v;
      }
    }

    // Use yearlyLimit if available, else sum of all monthly limits
    const resolvedYearlyLimit = yearlyLimit > 0 ? yearlyLimit : monthlyLimitSum;

    // Average monthly limit derived from yearly ÷ 12
    const avgMonthlyLimit = resolvedYearlyLimit > 0 ? Math.round(resolvedYearlyLimit / 12) : 0;

    items.push({
      category,
      categoryType: currentSection,
      monthlyLimit: avgMonthlyLimit,
      yearlyLimit: resolvedYearlyLimit,
      monthlyLimits, // per-month budget keyed "YYYY-MM"
      monthlySpent: 0,
      yearlySpent: 0,
      monthlyRemaining: avgMonthlyLimit,
      yearlyRemaining: resolvedYearlyLimit,
      note: row[1]?.trim() ?? '',
    });
  }

  await setCache(CACHE_KEY, items, 300);
  return items;
}
