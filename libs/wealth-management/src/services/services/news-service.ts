import { generateText, generateObject } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { executeSearch, type SearchResult } from './search-service';
import { z } from 'zod';

export interface AnalyzedNewsItem extends SearchResult {
  sentiment: 'Positive' | 'Negative' | 'Neutral';
  confidence: number;
  impactScore: number; // 0-100
  summary: string;
}

export interface NewsSentimentReport {
  topic: string;
  overallSentiment: 'BULLISH' | 'BEARISH' | 'NEUTRAL';
  counts: {
    positive: number;
    negative: number;
    neutral: number;
  };
  articles: AnalyzedNewsItem[];
  timestamp: string;
}

/**
 * Fetches and analyzes news for a specific topic or set of topics.
 */
export async function getAnalyzedNews(topic: string): Promise<NewsSentimentReport> {
  const model = getLanguageModel('github-gpt-4o');

  const currentYear = new Date().getFullYear();
  // 1. Fetch news
  const searchResponse = await executeSearch(`${topic} latest market news financial impact ${currentYear}`);
  const rawResults = searchResponse.results || [];

  if (rawResults.length === 0) {
    return {
      topic,
      overallSentiment: 'NEUTRAL',
      counts: { positive: 0, negative: 0, neutral: 0 },
      articles: [],
      timestamp: new Date().toISOString(),
    };
  }

  // 2. Analyze News with AI
  const prompt = `
    Analyze the following news articles related to "${topic}". 
    For each article, you MUST include the "index" provided in square brackets [i] in your response object to correctly map the analysis back to the original source.

    For each article, determine:
    1. Sentiment: Positive (Bullish), Negative (Bearish), or Neutral.
    2. Confidence: 0-100 score of how certain you are about this sentiment.
    3. Impact Score: 0-100 score of how much this news affects the market/topic.
    4. A 1-sentence summary of the core message.

    Articles:
    ${rawResults.map((r, i) => `[${i}] Title: ${r.title}\nSnippet: ${r.description}\nSource: ${r.source || new URL(r.url).hostname}`).join('\n\n')}

    Return the analysis in structured JSON format with a single top-level key "articles" containing the array of analyzed news items.

    Example format:
    {
      "articles": [
        { "index": 0, "sentiment": "Positive", "confidence": 90, "impactScore": 80, "summary": "..." }
      ]
    }
  `;

  const { object } = await generateObject({
    model,
    schema: z.object({
      articles: z.array(
        z.object({
          index: z.number(),
          sentiment: z.enum(['Positive', 'Negative', 'Neutral']),
          confidence: z.number().min(0).max(100),
          impactScore: z.number().min(0).max(100),
          summary: z.string(),
        }),
      ),
    }),
    prompt,
  });

  const analyzedArticles = object.articles.map((a, idx) => {
    // Fallback to current index if AI returns an invalid index
    const resultIndex =
      a.index !== undefined && a.index < rawResults.length ? a.index : idx < rawResults.length ? idx : 0;
    return {
      ...rawResults[resultIndex],
      sentiment: a.sentiment,
      confidence: a.confidence,
      impactScore: a.impactScore,
      summary: a.summary,
    };
  });

  const pos = analyzedArticles.filter((a) => a.sentiment === 'Positive').length;
  const neg = analyzedArticles.filter((a) => a.sentiment === 'Negative').length;
  const neut = analyzedArticles.filter((a) => a.sentiment === 'Neutral').length;

  let overall: 'BULLISH' | 'BEARISH' | 'NEUTRAL' = 'NEUTRAL';
  if (pos > neg && pos > neut) overall = 'BULLISH';
  if (neg > pos && neg > neut) overall = 'BEARISH';

  return {
    topic,
    overallSentiment: overall,
    counts: { positive: pos, negative: neg, neutral: neut },
    articles: analyzedArticles,
    timestamp: new Date().toISOString(),
  };
}
