export type TransactionCategory =
  | 'groceries'
  | 'restaurants'
  | 'fuel'
  | 'parking'
  | 'insurance'
  | 'utilities'
  | 'rent'
  | 'mortgage'
  | 'phone'
  | 'internet'
  | 'subscriptions'
  | 'gym'
  | 'medical'
  | 'pharmacy'
  | 'entertainment'
  | 'travel'
  | 'shopping'
  | 'gifts'
  | 'charity'
  | 'education'
  | 'books'
  | 'salary'
  | 'bonus'
  | 'investment income'
  | 'refund'
  | 'transfer'
  | 'fee'
  | 'interest'
  | 'tax'
  | 'other';

export interface TransactionCategoryMetadata {
  id: TransactionCategory;
  label: string;
  icon: string;
  color: string;
  type: 'income' | 'expense' | 'non-budget';
  description: string;
}

export interface CategoryTaggingRule {
  category: TransactionCategory;
  keywords: string[];
  merchants?: string[];
  autoTag?: boolean;
}
