// Predefined transaction tags — add new ones here as needed
export const TRANSACTION_TAGS = [
  'platinum',
  'platinum online',
  'uniq others',
  'uniq supermarket and transport',
] as const;

export type TransactionTag = (typeof TRANSACTION_TAGS)[number];
