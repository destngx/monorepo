import { NextResponse } from 'next/server';

const BASE_URL = 'https://api.fmarket.vn/res';

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
};

async function fmarketFetch(url: string, method = 'GET', body: any = null) {
  const fetchOptions: RequestInit = {
    method,
    headers: {
      ...fmarketHeaders,
      ...(body ? { 'Content-Type': 'application/json' } : {}),
    },
    credentials: 'omit',
  };

  if (body && (method === 'POST' || method === 'PUT')) {
    fetchOptions.body = JSON.stringify(body);
  }

  const response = await fetch(url, fetchOptions);
  return response.json();
}

export async function POST(request: Request) {
  try {
    const { action, params } = await request.json();

    switch (action) {
      case 'getProductsFilterNav': {
        const { page = 1, pageSize = 10, assetTypes = ['STOCK'], isMMFFund = false } = params;
        const data = await fmarketFetch(`${BASE_URL}/products/filter`, 'POST', {
          types: ['NEW_FUND', 'TRADING_FUND'],
          sortOrder: 'DESC',
          sortField: 'navTo12Months',
          isIpo: false,
          isMMFFund,
          fundAssetTypes: isMMFFund ? [] : assetTypes,
          page,
          pageSize,
        });
        return NextResponse.json(data);
      }

      case 'getIssuers': {
        const data = await fmarketFetch(`${BASE_URL}/issuers`, 'POST', {});
        return NextResponse.json(data);
      }

      case 'getBankInterestRates': {
        const data = await fmarketFetch(`${BASE_URL}/bank-interest-rate`, 'GET');
        return NextResponse.json(data);
      }

      case 'getProductDetails': {
        const { code } = params;
        const data = await fmarketFetch(`https://api.fmarket.vn/home/product/${code}`, 'GET');
        return NextResponse.json(data);
      }

      case 'getNavHistory': {
        const { productId, navPeriod = 'navToBeginning' } = params;
        const data = await fmarketFetch(`${BASE_URL}/product/get-nav-history`, 'POST', {
          isAllData: navPeriod === 'navToBeginning' ? 1 : 0,
          productId,
          navPeriod,
        });
        return NextResponse.json(data);
      }

      case 'getGoldPriceHistory': {
        const { fromDate, toDate } = params;
        const data = await fmarketFetch(`${BASE_URL}/get-price-gold-history`, 'POST', {
          fromDate,
          toDate,
          isAllData: false,
        });
        return NextResponse.json(data);
      }

      case 'getUsdRateHistory': {
        const { fromDate, toDate } = params;
        const data = await fmarketFetch(`${BASE_URL}/get-usd-rate-history`, 'POST', {
          fromDate,
          toDate,
          isAllData: false,
        });
        return NextResponse.json(data);
      }

      case 'getGoldProducts': {
        const data = await fmarketFetch(`${BASE_URL}/products/filter`, 'POST', {
          types: ['GOLD'],
          issuerIds: [],
          page: 1,
          pageSize: 100,
          fundAssetTypes: [],
          bondRemainPeriods: [],
          searchField: '',
        });
        return NextResponse.json(data);
      }

      default:
        return NextResponse.json({ error: 'Invalid action' }, { status: 400 });
    }
  } catch (error) {
    console.error('[FmarketAPI] Error:', error);
    return NextResponse.json(
      { error: 'Failed to fetch Fmarket data', details: error instanceof Error ? error.message : String(error) },
      { status: 500 },
    );
  }
}
