import { Account, AccountType } from '../types/account';
import { Transaction } from '../types/transaction';
import { BudgetItem } from '../types/budget';
import { parseDate } from '../utils/date';

// Helper to safely parse a number from the sheet.
// With UNFORMATTED_VALUE, numbers come as raw values (e.g., "1000000" not "1.000.000").
function num(value: string | undefined): number {
  if (!value || value.trim() === '') return 0;
  const n = parseFloat(value.replace(/,/g, ''));
  return isNaN(n) ? 0 : n;
}

// Header/help rows to skip
const JUNK_ACCOUNT_NAMES = ['help', 'accounts', 'account'];
const JUNK_PATTERNS = [
  /^you can track/i,
  /^accounts column/i,
  /^to get started/i,
];

function isJunkRow(name: string): boolean {
  if (!name || name.trim() === '') return true;
  const lower = name.trim().toLowerCase();
  if (JUNK_ACCOUNT_NAMES.includes(lower)) return true;
  return JUNK_PATTERNS.some(p => p.test(name));
}

/**
 * Accounts sheet layout (actual):
 * A: Name | B: Due Date | C: Goal Amount | D: Goal Progress (%) | E: Cleared Balance | F: Balance | G: Type | H: Note
 *
 * All balances are already in VND. Crypto accounts use sheet formulas to convert to VND.
 */
export function mapAccount(row: string[]): Account | null {
  const name = row[0]?.trim() || '';
  if (isJunkRow(name)) return null;

  const typeRaw = row[6]?.toLowerCase().trim() || '';
  // Skip the header row ("type" in the Type column)
  if (typeRaw === 'type' || typeRaw === 'date to pay') return null;

  // Due date: could be a serial number from UNFORMATTED_VALUE, plain text ("5, 30"), or empty
  let dueDate: string | null = row[1]?.trim() || null;
  if (dueDate && /^\d+$/.test(dueDate)) {
    const serial = parseInt(dueDate);
    if (serial > 1000) {
      // It's a serial date number — convert to readable date
      const epoch = new Date(1899, 11, 30);
      epoch.setDate(epoch.getDate() + serial);
      dueDate = `${epoch.getDate()}/${epoch.getMonth() + 1}/${epoch.getFullYear()}`;
    }
  }

  return {
    name,
    dueDate,
    goalAmount: num(row[2]) || null,
    goalProgress: row[3] ? parseFloat(row[3]) * 100 : null,
    clearedBalance: num(row[4]),
    balance: num(row[5]),
    type: (typeRaw as AccountType) || 'bank',
    currency: 'VND', // All values in the sheet are already in VND
    note: row[7]?.trim() || null,
  };
}

/**
 * Transactions sheet layout:
 * A: Account | B: Date | C: Ref# | D: Payee | E: Tags | F: Memo | G: Category | H: Cleared | I: Payment | J: Deposit | K-M: Balances
 */
export function mapTransaction(row: string[], index: number): Transaction | null {
  const accountName = row[0]?.trim() || '';
  if (isJunkRow(accountName)) return null;

  const category = row[6]?.trim() || 'Uncategorized';
  // Skip header row
  if (category === 'Category' || accountName === 'Account') return null;

  return {
    id: `row-${index}`,
    accountName,
    date: row[1] ? parseDate(row[1]) : new Date(),
    referenceNumber: row[2]?.trim() || null,
    payee: row[3]?.trim() || '',
    tags: row[4] ? row[4].split(',').map(s => s.trim()).filter(Boolean) : [],
    memo: row[5]?.trim() || null,
    category,
    cleared: row[7]?.toLowerCase() === 'y' || row[7]?.toLowerCase() === 'yes' || row[7] === '✓',
    payment: num(row[8]) || null,
    deposit: num(row[9]) || null,
    accountBalance: num(row[10]),
    clearedBalance: num(row[11]),
    runningBalance: num(row[12]),
  };
}

/**
 * Budget sheet layout:
 * A: Category | B: Monthly Limit | C: Yearly Limit | D: Monthly Spent | E: Yearly Spent | F: Monthly Remaining | G: Yearly Remaining
 */
const JUNK_BUDGET_CATEGORIES = ['help', 'income categories', 'expense categories', 'transfer categories'];

export function mapBudgetItem(row: string[]): BudgetItem | null {
  const category = row[0]?.trim() || '';
  if (!category) return null;

  const lower = category.toLowerCase();
  if (JUNK_BUDGET_CATEGORIES.includes(lower)) return null;
  // Skip pure uppercase section headers
  if (category === category.toUpperCase() && category.length > 3 && !/\d/.test(category)) return null;

  return {
    category,
    monthlyLimit: num(row[1]),
    yearlyLimit: num(row[2]),
    monthlySpent: num(row[3]),
    yearlySpent: num(row[4]),
    monthlyRemaining: num(row[5]),
    yearlyRemaining: num(row[6]),
  };
}

/**
 * Goals sheet layout (based on inspection):
 * E: Name (row[4])
 * G: Target Amount (row[6])
 * I: Current Amount (row[8])
 */
export function mapGoal(row: string[], index: number): any | null {
  const name = row[4]?.trim() || '';
  if (!name || name === 'Instructions' || name.includes('Insert more rows')) return null;

  // Amount parsing helper for VND strings like "2.000.000.000 ₫"
  const parseVND = (val: string | undefined) => {
    if (!val) return 0;
    const clean = val.replace(/[.₫\s]/g, '').replace(/,/g, '');
    return parseFloat(clean) || 0;
  };

  const targetAmount = parseVND(row[6]);
  const currentAmount = parseVND(row[8]);

  if (targetAmount === 0 && currentAmount === 0) return null;

  // Heuristic for Goal types
  let type: any = "SAVINGS_TARGET";
  let emoji = "🎯";

  const lower = name.toLowerCase();
  if (lower.includes('xe') || lower.includes('car')) {
    type = "PURCHASE_GOAL";
    emoji = "🚗";
  } else if (lower.includes('nhà') || lower.includes('house') || lower.includes('đất')) {
    type = "PURCHASE_GOAL";
    emoji = "🏠";
  } else if (lower.includes('học') || lower.includes('school')) {
    type = "SAVINGS_TARGET";
    emoji = "🎓";
  } else if (lower.includes('du lịch') || lower.includes('vacation')) {
    type = "PURCHASE_GOAL";
    emoji = "✈️";
  }

  return {
    id: `goal-${index}`,
    name,
    type,
    emoji,
    targetAmount,
    currentAmount,
    deadline: "TBD",
    status: (currentAmount / targetAmount) < 0.5 ? "AT_RISK" : "ON_TRACK",
    linkedAccountId: "sheet-goals",
    contributionType: "MANUAL",
    streakCount: 0,
    milestones: [],
    history: []
  };
}
