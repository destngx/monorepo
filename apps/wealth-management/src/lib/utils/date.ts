import { format, parse } from 'date-fns';

export function formatDate(date: Date | string): string {
  try {
    const d = typeof date === 'string' ? new Date(date) : date;
    if (isNaN(d.getTime())) return 'N/A';
    return format(d, 'MM/dd/yyyy');
  } catch {
    return 'N/A';
  }
}

export function parseDate(dateStr: string): Date {
  if (!dateStr || dateStr.trim() === '') return new Date();

  const trimmed = dateStr.trim();

  // Handle Google Sheets serial date numbers (UNFORMATTED_VALUE returns these)
  if (/^\d+(\.\d+)?$/.test(trimmed)) {
    const serial = parseFloat(trimmed);
    if (serial > 1000 && serial < 100000) {
      const epoch = new Date(1899, 11, 30);
      epoch.setDate(epoch.getDate() + serial);
      return epoch;
    }
  }

  // Try multiple common date string formats
  // Prioritize MM/dd/yyyy and M/d/yyyy since the sheet uses US-style dates
  const formats = [
    'MM/dd/yyyy',
    'M/d/yyyy',
    'dd/MM/yyyy',
    'd/M/yyyy',
    'MM/dd/yy',
    'M/d/yy',
    'dd/MM/yy',
    'd/M/yy',
    'yyyy-MM-dd'
  ];

  for (const fmt of formats) {
    try {
      const parsed = parse(trimmed, fmt, new Date());
      if (!isNaN(parsed.getTime())) {
        if (parsed.getFullYear() < 100) {
          parsed.setFullYear(parsed.getFullYear() + 2000);
        }
        return parsed;
      }
    } catch {
      // continue to next format
    }
  }

  // Last resort: native Date constructor
  const fallback = new Date(trimmed);
  if (!isNaN(fallback.getTime())) {
    if (fallback.getFullYear() < 100) {
      fallback.setFullYear(fallback.getFullYear() + 2000);
    }
    return fallback;
  }

  return new Date();
}

export function formatDateForSheets(date: Date): string {
  console.log("date", date);
  console.log("formatDate", format(date, 'MM/dd/yyyy'))
  return format(date, 'dd/MM/yyyy');
}
