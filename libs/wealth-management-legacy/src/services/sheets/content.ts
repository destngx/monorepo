import { readSheet } from './client';
import { getCached, setCache } from '@wealth-management/utils';

const PROMPTS_CACHE_KEY = 'sheets:prompts';
const KNOWLEDGE_CACHE_KEY = 'sheets:knowledge';
const CACHE_TTL = 3600; // 1 hour

type ContentMap = Record<string, string>;

/**
 * Fetches all prompts from the "Prompts" sheet and caches the result.
 * Sheet columns: domain | name | content
 * Returns a map keyed by "domain/name" → content.
 */
async function fetchAllPrompts(): Promise<ContentMap> {
  const cached = await getCached<ContentMap>(PROMPTS_CACHE_KEY);
  if (cached) return cached;

  const rows = await readSheet('Prompts!A2:C');
  const map: ContentMap = {};

  for (const row of rows) {
    const [domain, name, content] = row;
    if (domain && name && content) {
      map[`${domain}/${name}`] = content;
    }
  }

  await setCache(PROMPTS_CACHE_KEY, map, CACHE_TTL);
  return map;
}

/**
 * Fetches all knowledge entries from the "Knowledge" sheet and caches the result.
 * Sheet columns: name | content
 * Returns a map keyed by name → content.
 */
async function fetchAllKnowledge(): Promise<ContentMap> {
  const cached = await getCached<ContentMap>(KNOWLEDGE_CACHE_KEY);
  if (cached) return cached;

  const rows = await readSheet('Knowledge!A2:B');
  const map: ContentMap = {};

  for (const row of rows) {
    const [name, content] = row;
    if (name && content) {
      map[name] = content;
    }
  }

  await setCache(KNOWLEDGE_CACHE_KEY, map, CACHE_TTL);
  return map;
}

/**
 * Gets prompt content from the Google Sheet for a given domain/name.
 * Batch-fetches all prompts in a single API call (cached for 1h).
 */
export async function getPromptContent(domain: string, name: string): Promise<string | null> {
  const map = await fetchAllPrompts();
  return map[`${domain}/${name}`] ?? null;
}

/**
 * Gets knowledge content from the Google Sheet for a given name.
 * Batch-fetches all knowledge items in a single API call (cached for 1h).
 */
export async function getKnowledgeContent(name: string): Promise<string | null> {
  const map = await fetchAllKnowledge();
  return map[name] ?? null;
}

/**
 * Eagerly prefetches all prompts and knowledge content from Google Sheets.
 * Call this at app startup to warm the cache.
 * Throws if either fetch fails.
 */
export async function prefetchAllContent(): Promise<{ prompts: number; knowledge: number }> {
  const [prompts, knowledge] = await Promise.all([fetchAllPrompts(), fetchAllKnowledge()]);

  return {
    prompts: Object.keys(prompts).length,
    knowledge: Object.keys(knowledge).length,
  };
}
