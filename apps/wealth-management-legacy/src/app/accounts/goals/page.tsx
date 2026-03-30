export const dynamic = 'force-dynamic';
export const revalidate = false;
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

import GoalsPage from '@/features/goals/ui/page';

export default GoalsPage;
