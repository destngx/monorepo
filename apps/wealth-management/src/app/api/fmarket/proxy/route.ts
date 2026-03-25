import { NextResponse } from 'next/server';

export async function POST(request: Request) {
  try {
    const { url, method = 'GET', body, headers = {} } = await request.json();

    if (!url) {
      return NextResponse.json({ error: 'URL is required' }, { status: 400 });
    }

    const fmarketHeaders: Record<string, string> = {
      'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:148.0) Gecko/20100101 Firefox/148.0',
      Accept: 'application/json, text/plain, */*',
      'Accept-Language': 'en-US,en;q=0.9',
      'F-Language': 'vi',
      Origin: 'https://fmarket.vn',
      Referer: 'https://fmarket.vn/',
      'Sec-GPC': '1',
      'Sec-Fetch-Dest': 'empty',
      'Sec-Fetch-Mode': 'cors',
      'Sec-Fetch-Site': 'same-site',
      Priority: 'u=0',
      Pragma: 'no-cache',
      'Cache-Control': 'no-cache',
      ...headers,
    };

    const fetchOptions: RequestInit = {
      method,
      headers: fmarketHeaders,
      credentials: 'omit',
    };

    if (body && (method === 'POST' || method === 'PUT')) {
      fetchOptions.body = typeof body === 'string' ? body : JSON.stringify(body);
      if (!fmarketHeaders['Content-Type']) {
        fmarketHeaders['Content-Type'] = 'application/json';
      }
    }

    const response = await fetch(url, fetchOptions);
    const data = await response.json();

    return NextResponse.json(data);
  } catch (error) {
    console.error('[FmarketProxy] Error:', error);
    return NextResponse.json(
      { error: 'Failed to proxy request to Fmarket', details: error instanceof Error ? error.message : String(error) },
      { status: 500 },
    );
  }
}
