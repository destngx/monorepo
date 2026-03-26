export const dynamic = 'force-dynamic';
export const revalidate = false;
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

/**
 * Delegating page route
 * Maps app/budget to features/budget/ui/page
 */

import { BudgetPage } from '@wealth-management/features/budget/ui';

export default BudgetPage;
