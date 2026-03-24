import { NetworkError, isAppError, getErrorMessage } from '../../utils/errors';

export interface SearchResult {
  title: string;
  url: string;
  description: string;
  source?: string;
}

export interface SearchResponse {
  results?: SearchResult[];
  error?: string;
}

/**
 * Search Service: Executes web searches for real-time market intelligence.
 * Currently uses TAVILY_API_KEY.
 */
export async function executeSearch(query: string): Promise<SearchResponse> {
  const apiKey = process.env.TAVILY_API_KEY;
  if (!apiKey) {
    return { error: 'Tavily API key not configured' };
  }

  try {
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
      return { error: `Tavily API error: ${err}` };
    }

    const data = await response.json();
    return {
      results: data.results.map((r: any) => ({
        title: r.title,
        url: r.url,
        description: r.content,
        source: new URL(r.url).hostname.replace('www.', ''),
      })),
    };
  } catch (error) {
    const networkError = isAppError(error)
      ? error
      : new NetworkError('Search execution failed', {
          context: { source: 'tavily', query },
        });
    return { error: `Search execution failed: ${networkError.message}` };
  }
}
