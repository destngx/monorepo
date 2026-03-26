export const dynamic = 'force-dynamic';
export const revalidate = false;
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

/**
 * Delegating page route
 * Maps app/transactions to features/transactions/ui/page
 */

import { TransactionsPage } from '@wealth-management/features/transactions/ui';

export default TransactionsPage;
