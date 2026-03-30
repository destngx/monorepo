import { readSheet } from './client';

const HEADERS_TO_IGNORE = new Set(['categories', 'income categories', 'expense categories', 'non-budget categories']);

export interface Category {
  name: string;
  type: 'income' | 'expense' | 'non-budget';
}

export async function getCategories(): Promise<Category[]> {
  // Read column A of the Categories sheet
  const rows = await readSheet('Categories!A1:A153');

  const categories: Category[] = [];
  let currentGroup: 'income' | 'expense' | 'non-budget' | null = null;

  for (const row of rows) {
    const cell = row[0]?.trim();
    if (!cell) continue;

    const lower = cell.toLowerCase();

    // Check for headers to switch groups
    if (lower.includes('income categories')) {
      currentGroup = 'income';
      continue;
    } else if (lower.includes('expense categories')) {
      currentGroup = 'expense';
      continue;
    } else if (lower.includes('non-budget categories')) {
      currentGroup = 'non-budget';
      continue;
    } else if (lower === 'categories') {
      continue;
    }

    if (currentGroup && !HEADERS_TO_IGNORE.has(lower)) {
      categories.push({ name: cell, type: currentGroup });
    }
  }

  // Return unique categories, sorted by name
  const seen = new Set();
  return categories
    .filter((c) => {
      if (seen.has(c.name)) return false;
      seen.add(c.name);
      return true;
    })
    .sort((a, b) => a.name.localeCompare(b.name));
}
