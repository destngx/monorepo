export interface TagRule {
  pattern: string;
  tag: string;
  priority: number;
}

export interface CategoryRule {
  patterns: string[];
  category: string;
  priority: number;
}

export const TAG_RULES: TagRule[] = [
  { pattern: 'binance|kraken|coinbase|crypto', tag: 'crypto', priority: 10 },
  { pattern: 'amazon|ebay|shopping|retail', tag: 'shopping', priority: 9 },
  { pattern: 'uber|grab|taxi|lyft', tag: 'transport', priority: 9 },
  { pattern: 'starbucks|coffee|cafe', tag: 'coffee', priority: 8 },
  { pattern: 'netflix|spotify|hulu|disney', tag: 'subscription', priority: 8 },
  { pattern: 'gym|fitness|peloton', tag: 'fitness', priority: 8 },
  { pattern: 'restaurant|dining|food delivery', tag: 'dining', priority: 7 },
  { pattern: 'gas|fuel|shell|exxon', tag: 'fuel', priority: 7 },
];

export const CATEGORY_RULES: CategoryRule[] = [
  { patterns: ['safeway', 'kroger', 'whole foods', 'trader joe'], category: 'groceries', priority: 10 },
  { patterns: ['mcdonald', 'burger king', 'pizza hut', 'subway'], category: 'restaurants', priority: 10 },
  { patterns: ['shell', 'exxon', 'chevron', 'bp'], category: 'fuel', priority: 10 },
  { patterns: ['aetna', 'anthem', 'humana', 'wellcare'], category: 'insurance', priority: 10 },
  { patterns: ['verizon', 'at&t', 'tmobile', 'sprint'], category: 'phone', priority: 10 },
  { patterns: ['comcast', 'charter', 'fios', 'internet'], category: 'internet', priority: 10 },
  {
    patterns: ['netflix', 'spotify', 'hulu', 'disney', 'adobe', 'microsoft 365'],
    category: 'subscriptions',
    priority: 10,
  },
  { patterns: ['amazon', 'ebay', 'target', 'walmart', 'costco'], category: 'shopping', priority: 9 },
  { patterns: ['starbucks', 'dunkin', 'coffee'], category: 'restaurants', priority: 9 },
  { patterns: ['uber', 'lyft', 'grab', 'taxi'], category: 'travel', priority: 9 },
  { patterns: ['airline', 'airbnb', 'hotel', 'travel'], category: 'travel', priority: 9 },
  { patterns: ['gym', 'fitness', 'peloton', 'yoga'], category: 'gym', priority: 9 },
  { patterns: ['doctor', 'hospital', 'clinic', 'medical'], category: 'medical', priority: 9 },
  { patterns: ['cvs', 'walgreens', 'pharmacy'], category: 'pharmacy', priority: 9 },
  { patterns: ['movie', 'cinema', 'theater', 'entertainment'], category: 'entertainment', priority: 8 },
  { patterns: ['salary', 'paycheck', 'direct deposit'], category: 'salary', priority: 10 },
  { patterns: ['bonus', 'incentive'], category: 'bonus', priority: 10 },
  { patterns: ['dividend', 'interest', 'investment'], category: 'investment income', priority: 10 },
  { patterns: ['refund', 'reimbursement', 'reversal'], category: 'refund', priority: 10 },
  { patterns: ['transfer', 'xfer', 'between accounts'], category: 'transfer', priority: 10 },
  { patterns: ['fee', 'charge', 'charge back'], category: 'fee', priority: 10 },
  { patterns: ['tax', 'irs', 'withhold'], category: 'tax', priority: 10 },
];

export function suggestCategory(memo: string, payee: string): string {
  const searchText = `${memo} ${payee}`.toLowerCase();

  for (const rule of CATEGORY_RULES.sort((a, b) => b.priority - a.priority)) {
    for (const pattern of rule.patterns) {
      if (searchText.includes(pattern.toLowerCase())) {
        return rule.category;
      }
    }
  }

  return 'other';
}

export function extractTags(memo: string, payee: string): string[] {
  const searchText = `${memo} ${payee}`.toLowerCase();
  const tags = new Set<string>();

  for (const rule of TAG_RULES.sort((a, b) => b.priority - a.priority)) {
    const regex = new RegExp(rule.pattern, 'gi');
    if (regex.test(searchText)) {
      tags.add(rule.tag);
    }
  }

  return Array.from(tags);
}
