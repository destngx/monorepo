'use server';

import { readSheet, updateRow } from './client';
import { EmailNotification } from '@wealth-management/types/common';
import { getCached, setCache, invalidateCache } from '@wealth-management/utils';

const CACHE_KEY = 'notifications';
const SHEET_NAME = 'EmailNotifications';
const RANGE = `${SHEET_NAME}!A3:F`;

export async function getNotifications(forceFresh = false): Promise<EmailNotification[]> {
  if (!forceFresh) {
    const cached = await getCached<EmailNotification[]>(CACHE_KEY);
    if (cached) return cached;
  }

  const rows = await readSheet(RANGE);
  const notifications = rows
    .map((row, i) => {
      if (!row[0]) return null;
      return {
        id: row[0],
        timestamp: row[1],
        from: row[2],
        subject: row[3],
        content: row[4],
        status: (row[5]?.toLowerCase() === 'done' ? 'done' : 'pending') as 'pending' | 'done',
        rowNumber: i + 3,
      };
    })
    .filter((n): n is EmailNotification => n !== null);

  await setCache(CACHE_KEY, notifications, 300);
  return notifications;
}

export async function getPendingNotifications(): Promise<EmailNotification[]> {
  const all = await getNotifications();
  return all.filter((n) => n.status === 'pending');
}

export async function markNotificationDone(rowNumber: number) {
  // Status is column F (index 5)
  const range = `${SHEET_NAME}!F${rowNumber}`;
  await updateRow(range, ['done']);
  await invalidateCache(CACHE_KEY);
}
