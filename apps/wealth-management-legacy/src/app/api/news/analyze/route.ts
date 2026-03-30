import { getAnalyzedNews } from '@wealth-management/services/server';

export const maxDuration = 300;

export async function POST(req: Request) {
  const { topic } = (await req.json()) as { topic: string };
  try {
    const report = await getAnalyzedNews(topic);
    return Response.json(report);
  } catch (error) {
    console.error('Failed to get news analysis:', error);
    return Response.json({ error: 'Failed to fetch news analysis' }, { status: 500 });
  }
}
