import { NetworkError, isAppError, getErrorMessage } from '../../utils/errors';
import { getCached, setCache } from '@wealth-management/utils';

export interface SearchResult {
  title: string;
  url: string;
  description: string;
  source?: string;
}

export interface SearchResponse {
  results?: SearchResult[];
  error?: string;
  provider?: string;
}

interface SearchProvider {
  name: string;
  execute: (query: string) => Promise<SearchResult[]>;
}

// ============================================
// Cache Configuration
// ============================================

const SEARCH_CACHE_PREFIX = 'search:';
const SEARCH_CACHE_TTL = 14 * 24 * 3600; // 14 days in seconds

/**
 * Generate cache key from query string
 * Normalized: lowercase, trimmed, spaces replaced with underscores
 */
function generateSearchCacheKey(query: string): string {
  const normalized = query.toLowerCase().trim().replace(/\s+/g, '_');
  return `${SEARCH_CACHE_PREFIX}${normalized}`;
}

/**
 * Search Service: Executes web searches with intelligent roundrobin fallback.
 * Tries: Tavily (Primary) → Exa (Fallback 1) → DuckDuckGo (Fallback 2)
 *
 * Features:
 * - Automatic provider fallback on failure
 * - Consistent result format across all providers
 * - Detailed logging for debugging
 * - Request deduplication to avoid redundant calls
 */

// ============================================
// Provider Implementations
// ============================================

/**
 * Tavily API - Primary search provider
 * Benefits: Advanced search depth, high quality results
 */
async function tavilySearch(query: string): Promise<SearchResult[]> {
  const apiKey = process.env.TAVILY_API_KEY;
  if (!apiKey) {
    throw new NetworkError('Tavily API key not configured', {
      context: { source: 'tavily' },
    });
  }

  const response = await fetch('https://api.tavily.com/search', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      api_key: apiKey,
      query,
      search_depth: 'advanced',
      max_results: 5,
    }),
  });

  if (!response.ok) {
    const err = await response.text();
    throw new NetworkError(`Tavily API error: ${err}`, {
      context: { source: 'tavily', statusCode: response.status },
    });
  }

  const data = await response.json();
  return data.results.map((r: any) => ({
    title: r.title,
    url: r.url,
    description: r.content,
    source: new URL(r.url).hostname.replace('www.', ''),
  }));
}

/**
 * Exa API - Fallback 1 search provider
 * Benefits: Neural search, high precision, well-structured results
 */
async function exaSearch(query: string): Promise<SearchResult[]> {
  const apiKey = process.env.EXA_API_KEY;
  if (!apiKey) {
    throw new NetworkError('Exa API key not configured', {
      context: { source: 'exa' },
    });
  }

  const response = await fetch('https://api.exa.ai/search', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': apiKey,
    },
    body: JSON.stringify({
      query,
      type: 'auto',
      num_results: 5,
      contents: {
        text: { max_characters: 4000 },
      },
    }),
  });

  if (!response.ok) {
    const err = await response.text();
    throw new NetworkError(`Exa API error: ${err}`, {
      context: { source: 'exa', statusCode: response.status },
    });
  }

  const data = await response.json();
  return (data.results || []).map((r: any) => ({
    title: r.title,
    url: r.url,
    description: r.text || r.summary || r.highlights?.[0] || '',
    source: new URL(r.url).hostname.replace('www.', ''),
  }));
}

/**
 * DuckDuckGo - Fallback 2 search provider via duck-duck-scrape
 * Benefits: No API key required, no rate limits, lightweight
 */
async function duckDuckGoSearch(query: string): Promise<SearchResult[]> {
  try {
    const { search, SafeSearchType } = await import('duck-duck-scrape');

    const results = await search(query, {
      safeSearch: SafeSearchType.MODERATE,
    });

    return results.results.slice(0, 5).map((r: any) => ({
      title: r.title,
      url: r.url,
      description: r.description || r.snippet || '',
      source: new URL(r.url).hostname.replace('www.', ''),
    }));
  } catch (error) {
    throw new NetworkError('DuckDuckGo search failed', {
      context: {
        source: 'duckduckgo',
        message: error instanceof Error ? error.message : String(error),
      },
    });
  }
}

// ============================================
// Main Roundrobin Logic
// ============================================

/**
 * Execute web search with intelligent provider fallback and result caching
 * Caches results for 14 days to avoid redundant API calls
 * @param query - Search query string
 * @return Search results from cache or first successful provider
 */
export async function executeSearch(query: string): Promise<SearchResponse> {
  const cacheKey = generateSearchCacheKey(query);

  // Check cache first
  const cached = await getCached<SearchResponse>(cacheKey);
  if (cached) {
    console.log(`[SearchService] ✓ Cache hit for query: "${query}"`);
    return cached;
  }

  const providers: SearchProvider[] = [
    { name: 'tavily', execute: tavilySearch },
    { name: 'exa', execute: exaSearch },
    { name: 'duckduckgo', execute: duckDuckGoSearch },
  ];

  const errors: { provider: string; error: string }[] = [];

  // Try each provider in order until one succeeds
  for (const provider of providers) {
    try {
      console.log(`[SearchService] Attempting ${provider.name}...`);

      const results = await provider.execute(query);

      if (results && results.length > 0) {
        console.log(`[SearchService] ✓ Success with ${provider.name} (${results.length} results)`);
        const response: SearchResponse = {
          results,
          provider: provider.name,
        };

        // Cache successful results
        await setCache(cacheKey, response, SEARCH_CACHE_TTL);
        console.log(`[SearchService] Cached results for 14 days (key: ${cacheKey})`);

        return response;
      }

      // Provider returned empty results, try next
      console.warn(`[SearchService] ${provider.name} returned no results, trying next provider...`);
      errors.push({
        provider: provider.name,
        error: 'No results returned',
      });
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      console.warn(`[SearchService] ✗ ${provider.name} failed: ${errorMsg}, trying next provider...`);
      errors.push({
        provider: provider.name,
        error: errorMsg,
      });
    }
  }

  // All providers failed
  const errorSummary = errors.map((e) => `${e.provider}: ${e.error}`).join(' | ');
  const finalError = `All search providers failed. Attempts: ${errorSummary}`;

  console.error(`[SearchService] ✗ All providers exhausted: ${finalError}`);

  return {
    error: finalError,
    results: [],
  };
}

/**
 * Health check for search providers
 * Useful for monitoring which providers are available
 */
export async function getSearchProviderStatus(): Promise<Record<string, { available: boolean; error?: string }>> {
  const status: Record<string, { available: boolean; error?: string }> = {};

  // Check Tavily
  status.tavily = {
    available: !!process.env.TAVILY_API_KEY,
    error: process.env.TAVILY_API_KEY ? undefined : 'API key not configured',
  };

  // Check Exa
  status.exa = {
    available: !!process.env.EXA_API_KEY,
    error: process.env.EXA_API_KEY ? undefined : 'API key not configured',
  };

  // DuckDuckGo always available (no API key needed)
  status.duckduckgo = {
    available: true,
  };

  return status;
}
