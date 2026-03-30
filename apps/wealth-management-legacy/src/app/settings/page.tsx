export const dynamic = 'force-dynamic';
export const revalidate = false;
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

import SettingsPage from '@/features/settings/ui/page';

export default SettingsPage;
