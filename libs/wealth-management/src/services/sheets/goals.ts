import { readSheet } from './client';
import { mapGoal } from './mappers';
import { Goal } from '@wealth-management/types/goals';
import { getCached, setCache } from '@wealth-management/utils';

const CACHE_KEY = 'goals';

export async function getGoals(forceFresh = false): Promise<Goal[]> {
  if (!forceFresh) {
    const cached = await getCached<Goal[]>(CACHE_KEY);
    if (cached) return cached;
  }

  // Reading the first 50 rows of the Goals sheet
  const rows = await readSheet('Goals!A1:Z50');

  const goals = rows.map((row, idx) => mapGoal(row, idx)).filter((g): g is Goal => g !== null && g.name !== '');

  await setCache(CACHE_KEY, goals, 300); // 5 min cache
  return goals;
}
