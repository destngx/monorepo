import { NextResponse } from 'next/server';
import { getOrSetCache, CACHE_KEYS, CACHE_TTL } from '@/shared/cache';

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

/**
 * Cached Fmarket API actions
 */
const actions = {
  getProductsFilterNav: (params: any) => {
    const { page = 1, pageSize = 10, assetTypes = ['STOCK'], isMMFFund = false } = params;
    return fmarketFetch(`${BASE_URL}/products/filter`, 'POST', {
      types: ['NEW_FUND', 'TRADING_FUND'],
      sortOrder: 'DESC',
      sortField: 'navTo12Months',
      isIpo: false,
      isMMFFund,
      fundAssetTypes: isMMFFund ? [] : assetTypes,
      page,
      pageSize,
    });
  },

  getIssuers: () => fmarketFetch(`${BASE_URL}/issuers`, 'POST', {}),

  getBankInterestRates: () => fmarketFetch(`${BASE_URL}/bank-interest-rate`, 'GET'),

  getProductDetails: (params: any) => {
    const { code } = params;
    return fmarketFetch(`https://api.fmarket.vn/home/product/${code}`, 'GET');
  },

  getNavHistory: (params: any) => {
    const { productId, navPeriod = 'navToBeginning' } = params;
    return fmarketFetch(`${BASE_URL}/product/get-nav-history`, 'POST', {
      isAllData: 1,
      productId,
      navPeriod,
    });
  },

  getGoldPriceHistory: (params: any) => {
    const { fromDate, toDate } = params;
    return fmarketFetch(`${BASE_URL}/get-price-gold-history`, 'POST', {
      fromDate,
      toDate,
      isAllData: false,
    });
  },

  getUsdRateHistory: (params: any) => {
    const { fromDate, toDate } = params;
    return fmarketFetch(`${BASE_URL}/get-usd-rate-history`, 'POST', {
      fromDate,
      toDate,
      isAllData: false,
    });
  },

  getGoldProducts: () =>
    fmarketFetch(`${BASE_URL}/products/filter`, 'POST', {
      types: ['GOLD'],
      issuerIds: [],
      page: 1,
      pageSize: 100,
      fundAssetTypes: [],
      bondRemainPeriods: [],
      searchField: '',
    }),
};

export async function POST(request: Request) {
  try {
    const { action, params = {} } = await request.json();
    const { searchParams } = new URL(request.url);
    const forceFresh = searchParams.get('force') === 'true';

    if (!(action in actions)) {
      return NextResponse.json({ error: 'Invalid action' }, { status: 400 });
    }

    // Use centralized getOrSetCache with action-specific keys
    const cacheKey = `fmarket:${action}:${JSON.stringify(params)}`;
    const data = await getOrSetCache(
      cacheKey,
      () => (actions as any)[action](params),
      CACHE_TTL.MARKET_DATA,
      forceFresh,
    );

    return NextResponse.json(data);
  } catch (error) {
    console.error('[FmarketAPI] Error:', error);
    return NextResponse.json(
      { error: 'Failed to fetch Fmarket data', details: error instanceof Error ? error.message : String(error) },
      { status: 500 },
    );
  }
}
