import { NextRequest, NextResponse } from 'next/server';
import { executeSearch } from '@wealth-management/services/services/search-service';
import { generateObject } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { z } from 'zod';

export async function POST(req: NextRequest) {
  try {
    const { symbol, modelId } = await req.json();

    if (!symbol) {
      return NextResponse.json({ error: 'Symbol is required' }, { status: 400 });
    }

    const searchQuery = `"${symbol}" stock profile company details sector industry exchange vietnam us`;
    const searchResponse = await executeSearch(searchQuery);

    if (searchResponse.error) {
      return NextResponse.json({ error: searchResponse.error }, { status: 500 });
    }

    const searchContext = searchResponse.results?.map((r) => `${r.title}\n${r.description}`).join('\n\n') || '';

    const model = getLanguageModel(modelId || 'github-gpt-4.1');
    const { object } = await generateObject({
      model,
      schema: z.object({
        symbol: z.string(),
        fullName: z.string(),
        description: z.string(),
        sector: z.string(),
        industry: z.string(),
        market: z.enum(['VN', 'US']),
      }),
      prompt: `
        Analyze the search results for the ticker symbol: ${symbol}.
        Extract the company profile, sector, industry, and determine if it belongs to the target Vietnam (VN) or US market.
        Provide a concise 1-2 sentence description in Vietnamese.
        
        Return the result as JSON with the following structure:
        {
          "symbol": string,
          "fullName": string,
          "description": string,
          "sector": string,
          "industry": string,
          "market": "VN" | "US"
        }

        Search Context:
        ${searchContext}
      `,
    });

    return NextResponse.json(object);
  } catch (error: any) {
    const statusCode = error.statusCode || 500;
    return NextResponse.json({ error: error.message }, { status: statusCode });
  }
}
