export const dynamic = 'force-dynamic';
export const revalidate = false;
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

import LoansPage from '@/features/loans/ui/page';

export default LoansPage;
