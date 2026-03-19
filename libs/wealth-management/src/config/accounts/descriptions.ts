import type { AccountType } from './types';

export interface AccountTypeDescription {
  usage: string;
  guidelines: string[];
  examples: string[];
}

export const ACCOUNT_TYPE_DESCRIPTIONS: Record<AccountType, AccountTypeDescription> = {
  'active use': {
    usage: 'Primary accounts for regular daily and weekly transactions',
    guidelines: [
      'Keep sufficient balance for immediate needs (2-4 weeks of expenses)',
      'Monitor regularly for fraud and unauthorized transactions',
      'Set up alerts for low balance conditions',
    ],
    examples: ['Daily spending checking account', 'Primary salary deposit account', 'Frequent debit card account'],
  },
  'rarely use': {
    usage: 'Secondary accounts accessed infrequently, kept for specific purposes',
    guidelines: [
      'Review quarterly for activity',
      'Maintain minimal balance if required by bank',
      'Update emergency contact information',
    ],
    examples: [
      'Old bank account kept for legacy transfers',
      'Secondary savings account',
      'Regional bank branch account',
    ],
  },
  'long holding': {
    usage: 'Savings and investment accounts for medium to long-term financial goals',
    guidelines: [
      'Establish clear target date and goal amount',
      'Minimize withdrawals to preserve compound growth',
      'Review annually against inflation and interest rates',
      'Consider tax-advantaged instruments where available',
    ],
    examples: [
      'High-yield savings account for 1-2 year goals',
      'Term deposit or fixed income account',
      'Education savings account',
    ],
  },
  deprecated: {
    usage: 'Archived accounts no longer in active use but kept for record-keeping',
    guidelines: [
      'Keep records for historical analysis and tax purposes',
      'Verify account closure status with financial institution',
      'Archive related transaction records',
      'Do not transfer funds to or from these accounts',
    ],
    examples: ['Closed checking account', 'Transferred savings account', 'Old employer 401k account (rolled over)'],
  },
  'negative active use': {
    usage: 'Credit facilities and liabilities actively being repaid',
    guidelines: [
      'Track minimum payment due dates religiously',
      'Monitor interest rates and fees',
      'Prioritize by interest rate (highest first for repayment)',
      'Watch for early payoff penalties on loans',
    ],
    examples: ['Credit card with outstanding balance', 'Personal loan', 'Line of credit'],
  },
  bank: {
    usage: 'Traditional banking accounts with FDIC/equivalent insurance protection',
    guidelines: [
      'Verify FDIC/equivalent insurance coverage ($250k limits)',
      'Consolidate if balance exceeds insurance limits',
      'Review fee structures quarterly',
      'Enable overdraft protection if needed',
    ],
    examples: ['Checking account', 'Savings account', 'Money market account'],
  },
  crypto: {
    usage: 'Digital asset holdings and cryptocurrency exchange accounts',
    guidelines: [
      'Implement strong security (2FA, hardware wallet consideration)',
      'Track tax lot acquisition for capital gains reporting',
      'Monitor exchange rate fluctuations',
      'Understand staking and DeFi protocol risks',
      'Use reputable, regulated exchanges',
    ],
    examples: ['Binance account with crypto holdings', 'MetaMask wallet with ETH', 'Kraken trading account'],
  },
  cash: {
    usage: 'Physical cash holdings for emergency access and offline funds',
    guidelines: [
      'Keep in secure location (safe, safe deposit box)',
      'Limit to 1-3 months emergency expenses',
      'Account for inflation impact on purchasing power',
      'Periodically verify physical notes remain secure',
    ],
    examples: ['Emergency cash in home safe', 'Safe deposit box cash', 'Physical currency emergency fund'],
  },
  investment: {
    usage: 'Brokerage and investment accounts for wealth building',
    guidelines: [
      'Understand tax implications (LTCG, short-term gains)',
      'Diversify across asset classes',
      'Monitor expense ratios and trading costs',
      'Rebalance portfolio quarterly/annually',
      'Use tax-advantaged accounts when available',
    ],
    examples: [
      'Brokerage account with stocks/ETFs',
      'Mutual fund account',
      'Retirement account (IRA, 401k equivalent)',
    ],
  },
};
