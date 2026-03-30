export const dynamic = 'force-dynamic';
export const revalidate = false;
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

/**
 * Delegating page route
 * Maps app/investments to features/investments/ui/page
 */

import { InvestmentsPage } from '@/features/investments/ui';

export default InvestmentsPage;
