import { Transaction } from '../types/transactions';

export interface CardCashbackRule {
  tag: string;
  rate: number;
  capMonthly: number;
  name: string;
}
export const SACOMBANK_CASHBACK_CREDIT = 40;
export const SACOMBANK_CASHBACK_RULES: Record<string, CardCashbackRule[]> = {
  'Sacombank Visa Platinum Cashback': [
    { tag: 'platinum online', rate: 5, capMonthly: 600000, name: 'Online' },
    { tag: 'platinum overseas', rate: 3, capMonthly: 600000, name: 'Overseas' },
    { tag: 'platinum', rate: 0.5, capMonthly: 600000, name: 'Others' },
  ],
  'Sacombank Visa UNIQ Platinum': [
    { tag: 'uniq supermarket and transport', rate: 20, capMonthly: 300000, name: 'Supermarket & Transport' },
    { tag: 'uniq others', rate: 0.5, capMonthly: 300000, name: 'Others' },
  ],
};

// Total cap for UNIQ is 600,000 (300k category + 300k others)
// Total cap for Platinum Cashback is 600,000 shared

export function calculateTransactionCashback(transaction: Transaction): { amount: number; ruleName: string } {
  const accountName = transaction.accountName.toLowerCase();
  const tags = transaction.tags.map((t) => t.toLowerCase());
  const payment = transaction.payment || 0;

  if (payment <= 0) return { amount: 0, ruleName: 'None' };

  // Identify card
  let cardName = '';
  if (accountName.includes('uniq') || tags.some((t) => t.includes('uniq'))) {
    cardName = 'Sacombank Visa UNIQ Platinum';
  } else if (accountName.includes('platinum') || tags.some((t) => t.includes('platinum'))) {
    cardName = 'Sacombank Visa Platinum Cashback';
  }

  if (!cardName) return { amount: 0, ruleName: 'None' };

  const rules = SACOMBANK_CASHBACK_RULES[cardName];
  if (!rules) return { amount: 0, ruleName: 'None' };

  // Match by tag first (most precise)
  for (const rule of rules) {
    if (tags.includes(rule.tag)) {
      return {
        amount: Math.round((payment * rule.rate) / 100),
        ruleName: rule.name,
      };
    }
  }

  // Fallback to card identity if tags are generic but card is known
  const fallbackRule = rules.find((r) => r.name === 'Others');
  if (fallbackRule) {
    return {
      amount: Math.round((payment * fallbackRule.rate) / 100),
      ruleName: fallbackRule.name,
    };
  }

  return { amount: 0, ruleName: 'None' };
}

export function getCreditCardSummary(transactions: Transaction[], accounts?: any[]) {
  const knownCards = [
    {
      name: 'Sacombank Visa UNIQ Platinum',
      bank: 'Sacombank',
      pattern: /uniq/i,
      tagKeyword: 'uniq',
      defaultLimit: 40000000,
      expiry: '08/2029',
    },
    {
      name: 'Sacombank Visa Platinum Cashback',
      bank: 'Sacombank',
      pattern: /platinum/i,
      tagKeyword: 'platinum',
      defaultLimit: 40000000,
      expiry: '11/2028',
    },
  ];

  // Try to find a single shared account first
  const sharedAccount = accounts?.find(
    (a) => a.name === 'Credit Card Sacombank' && a.type?.toLowerCase().includes('negative'),
  );

  // First, calculate individual card stats
  const cardResults = knownCards.map((cardInfo) => {
    // Find specific account or use shared
    const realAccount =
      accounts?.find((a) => a.name.match(cardInfo.pattern) && a.type?.toLowerCase().includes('negative')) ||
      sharedAccount;

    const accountKey =
      realAccount?.name || (cardInfo.tagKeyword === 'uniq' ? 'Credit Card Sacombank UNIQ' : 'Credit Card Sacombank');
    const limit = realAccount?.goalAmount || cardInfo.defaultLimit;
    const balance = realAccount?.clearedBalance || 0; // Negative balance for credit
    const totalSpent = Math.abs(balance);

    // Filter transactions correctly for the shared account logic
    const cardTransactions = transactions.filter((t) => {
      const isAccount = t.accountName === accountKey;
      const lowerTags = (t.tags || []).map((tag) => tag.toLowerCase());
      const hasCardTag = lowerTags.some((tag) => tag.includes(cardInfo.tagKeyword));

      if (isAccount) {
        // If we share the account, we MUST avoid the other card's tags.
        // If NO card tags are present, we assign to the primary card (Platinum) to avoid double counting.
        const otherKeyword = cardInfo.tagKeyword === 'uniq' ? 'platinum' : 'uniq';
        const hasOtherCardTag = lowerTags.some((tag) => tag.includes(otherKeyword));

        if (hasCardTag) return true;
        if (hasOtherCardTag) return false;

        // Default untagged transactions in shared account to Platinum Cashback
        return cardInfo.tagKeyword === 'platinum';
      }

      return hasCardTag;
    });

    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);

    const thisCycleTransactions = cardTransactions.filter((t) => {
      const d = new Date(t.date);
      return d >= thirtyDaysAgo;
    });

    const cashbackByRule: Record<string, number> = {};
    thisCycleTransactions.forEach((t) => {
      const result = calculateTransactionCashback(t);
      if (result.amount > 0) {
        cashbackByRule[result.ruleName] = (cashbackByRule[result.ruleName] || 0) + result.amount;
      }
    });

    let cycleCashback = 0;
    const rules = SACOMBANK_CASHBACK_RULES[cardInfo.name] || [];
    rules.forEach((rule) => {
      cycleCashback += Math.min(cashbackByRule[rule.name] || 0, rule.capMonthly);
    });
    cycleCashback = Math.min(cycleCashback, 600000);

    // Calculate YTD Refund
    const currentYear = new Date().getFullYear();
    const ytdRefund = cardTransactions.reduce((sum, t) => {
      const isCashbackPayee = t.payee.toLowerCase().includes('cashback');
      const isRefundCategory = t.category.toLowerCase().replace(/\s/g, '') === 'refunds/reimbursements';
      const isCashback = (t.deposit || 0) > 0 && isCashbackPayee && isRefundCategory;

      if (!isCashback) return sum;

      const effectiveDate = getEffectiveDate(t);
      if (effectiveDate.getFullYear() === currentYear) {
        return sum + (t.deposit || 0);
      }
      return sum;
    }, 0);

    const CARD_FEE_ANNUAL = 599000;
    // Estimate years active (simple: min 1)
    const dates = cardTransactions.map((t) => new Date(t.date).getTime());
    const minDateNode = dates.length > 0 ? Math.min(...dates) : new Date().getTime();
    const yearsActive = Math.max(1, Math.ceil((new Date().getTime() - minDateNode) / (365 * 24 * 60 * 60 * 1000)));
    const totalFees = yearsActive * CARD_FEE_ANNUAL;

    return {
      ...cardInfo,
      accountKey,
      limit,
      totalUsage: totalSpent,
      spentThisMonth: thisCycleTransactions.reduce((sum, t) => {
        if (t.category === '[Beginning Balance]') return sum;
        return sum + (t.payment || 0);
      }, 0),
      estimatedCashback: cycleCashback,
      lifetimeCashback: ytdRefund, // Keeping the property name for UI compatibility, but it's now YTD
      totalFees,
      netEarn: ytdRefund - totalFees,
      transactionCount: thisCycleTransactions.length,
      history: getMonthlyHistory(cardTransactions),
      expiry: cardInfo.expiry,
    };
  });

  // Then, group by bank and account
  const banks: any[] = [];
  cardResults.forEach((card) => {
    let bank = banks.find((b) => b.name === card.bank);
    if (!bank) {
      bank = { name: card.bank, accounts: [] };
      banks.push(bank);
    }

    let account = bank.accounts.find((a: any) => a.name === card.accountKey);
    if (!account) {
      account = {
        name: card.accountKey,
        limit: card.limit,
        totalUsage: card.totalUsage,
        utilization: (card.totalUsage / card.limit) * 100,
        remainingLimit: card.limit - card.totalUsage,
        isShared: card.accountKey === 'Credit Card Sacombank',
        totalRefund: 0,
        totalFees: 0,
        totalEarn: 0,
        cards: [],
      };
      bank.accounts.push(account);
    }

    account.totalRefund += card.lifetimeCashback;
    account.totalFees += card.totalFees;
    account.totalEarn += card.netEarn;

    // Shared cap for Sacombank current cycle estimation
    account.estimatedCashback = Math.min((account.estimatedCashback || 0) + card.estimatedCashback, 600000);

    account.cards.push({
      name: card.name,
      spentThisMonth: card.spentThisMonth,
      estimatedCashback: card.estimatedCashback,
      lifetimeCashback: card.lifetimeCashback,
      totalFees: card.totalFees,
      netEarn: card.netEarn,
      transactionCount: card.transactionCount,
      history: card.history,
      expiry: card.expiry,
      accountKey: card.accountKey,
    });
  });

  return banks;
}

function getMonthlyHistory(transactions: Transaction[]) {
  const months: Record<string, { spent: number; calculatedCashback: number; actualRefund: number }> = {};

  transactions.forEach((t) => {
    const d = new Date(t.date);
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    if (!months[key]) months[key] = { spent: 0, calculatedCashback: 0, actualRefund: 0 };

    // Standard spending (payment) - ignore Beginning Balance
    const payment = t.category === '[Beginning Balance]' ? 0 : t.payment || 0;
    months[key].spent += payment;

    // Potential cashback based on rules
    const cb = calculateTransactionCashback(t);
    months[key].calculatedCashback += cb.amount;

    // Actual Refund received (User's Formula logic)
    const isCashbackPayee = t.payee.toLowerCase().includes('cashback');
    const isRefundCategory = t.category.toLowerCase().replace(/\s/g, '') === 'refunds/reimbursements';
    const isRefund = (t.deposit || 0) > 0 && isCashbackPayee && isRefundCategory;

    if (isRefund) {
      const effectiveDate = getEffectiveDate(t);
      const targetMonthKey = `${effectiveDate.getFullYear()}-${String(effectiveDate.getMonth() + 1).padStart(2, '0')}`;

      // Ensure the target month bucket exists
      if (!months[targetMonthKey]) {
        months[targetMonthKey] = { spent: 0, calculatedCashback: 0, actualRefund: 0 };
      }

      months[targetMonthKey].actualRefund += t.deposit || 0;
    }
  });

  return Object.entries(months)
    .sort((a, b) => b[0].localeCompare(a[0]))
    .slice(0, 36) // Extended to 3 years for annual chart tracking
    .map(([month, stats]) => ({
      month,
      spent: stats.spent,
      cashback: Math.min(stats.calculatedCashback, 600000),
      actualRefund: stats.actualRefund,
      efficiency: stats.spent > 0 ? (stats.actualRefund / stats.spent) * 100 : 0,
    }));
}

export function getEffectiveDate(t: Transaction): Date {
  const txDate = new Date(t.date);

  const isRefundCategory = t.category.toLowerCase().replace(/\s/g, '') === 'refunds/reimbursements';
  const isRefund = (t.deposit || 0) > 0 && isRefundCategory;

  if (!t.memo || !isRefund) return txDate;

  const memoStr = String(t.memo).replace(/['"]/g, '').trim();
  const memoNum = Number(memoStr);

  if (!isNaN(memoNum) && memoNum > 40000 && memoNum < 60000) {
    const dateObj = new Date(1899, 11, 30);
    dateObj.setDate(dateObj.getDate() + Math.floor(memoNum));
    console.log(`[Cashback Utils] getEffectiveDate: Converted ${memoNum} to ${dateObj.toISOString()}`);
    return dateObj;
  }

  const matchMDY = memoStr.match(/(0?[1-9]|1[0-2])\/(0?[1-9]|[12]\d|3[01])\/(\d{2}|\d{4})/);
  const matchMMYYYY = memoStr.match(/(0?[1-9]|1[0-2])[-/](20\d{2})/);
  const matchYYYYMM = memoStr.match(/(20\d{2})[-/](0?[1-9]|1[0-2])/);

  if (matchMDY) {
    const yStr = matchMDY[3].length === 2 ? `20${matchMDY[3]}` : matchMDY[3];
    const dateObj = new Date(Number(yStr), Number(matchMDY[1]) - 1, 1);
    console.log(`[Cashback Utils] getEffectiveDate: Parsed MDY ${memoStr} to ${dateObj.toISOString()}`);
    return dateObj;
  } else if (matchMMYYYY) {
    const dateObj = new Date(Number(matchMMYYYY[2]), Number(matchMMYYYY[1]) - 1, 1);
    console.log(`[Cashback Utils] getEffectiveDate: Parsed MMYYYY ${memoStr} to ${dateObj.toISOString()}`);
    return dateObj;
  } else if (matchYYYYMM) {
    const dateObj = new Date(Number(matchYYYYMM[1]), Number(matchYYYYMM[2]) - 1, 1);
    console.log(`[Cashback Utils] getEffectiveDate: Parsed YYYYMM ${memoStr} to ${dateObj.toISOString()}`);
    return dateObj;
  }

  return txDate;
}
