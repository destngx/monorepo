const dateFormatter = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  timeZone: 'UTC',
});

export const formatDate = (date: Date): string => dateFormatter.format(date);

const WORDS_PER_MINUTE = 220;

export const readingTime = (text: string): number =>
  Math.max(1, Math.ceil(text.trim().split(/\s+/).filter(Boolean).length / WORDS_PER_MINUTE));
