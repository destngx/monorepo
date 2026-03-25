export const BASE_URL = 'https://api.fmarket.vn/res';
const PROXY_URL = '/api/fmarket/proxy';

const proxyFetch = async (url: string, method = 'GET', body: any = null, customHeaders: any = {}) => {
  const response = await fetch(PROXY_URL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      url,
      method,
      body,
      headers: customHeaders,
    }),
  });
  const data = await response.json();
  if (url.includes('get-nav-history')) {
    console.log(`NAV History Response for ${url}:`, data?.data?.slice(0, 2));
  }
  return data;
};

export const fmarketApi = {
  getProductsFilterNav: async (page = 1, pageSize = 10, assetTypes = ['STOCK'], isMMFFund = false) => {
    return proxyFetch(`${BASE_URL}/products/filter`, 'POST', {
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

  getProductsFilterAnnual: async (page = 1, pageSize = 10, assetTypes = ['STOCK']) => {
    return proxyFetch(`${BASE_URL}/products/filter`, 'POST', {
      types: ['NEW_FUND', 'TRADING_FUND'],
      isNotSafefy: false,
      sortOrder: 'desc',
      sortField: 'annualizedReturn36Months',
      fundAssetTypes: assetTypes,
      page,
      pageSize,
    });
  },

  getIssuers: async () => {
    return proxyFetch(`${BASE_URL}/issuers`, 'POST', {});
  },

  getBankInterestRates: async () => {
    return proxyFetch(`${BASE_URL}/bank-interest-rate`, 'GET');
  },

  getProductDetails: async (code: string) => {
    return proxyFetch(`https://api.fmarket.vn/home/product/${code}`, 'GET');
  },

  getNavHistory: async (productId: number, navPeriod = 'navToBeginning') => {
    // Fmarket navPeriod expected values:
    // navTo24Months, navToBeginning, navTo12Months, navTo6Months, navTo36Months, navTo60Months, navToYtd
    return proxyFetch(`${BASE_URL}/product/get-nav-history`, 'POST', {
      isAllData: navPeriod === 'navToBeginning' ? 1 : 0,
      productId,
      navPeriod,
    });
  },

  getGoldPriceHistory: async (fromDate: string, toDate: string) => {
    return proxyFetch(`${BASE_URL}/get-price-gold-history`, 'POST', { fromDate, toDate, isAllData: false });
  },

  getUsdRateHistory: async (fromDate: string, toDate: string) => {
    return proxyFetch(`${BASE_URL}/get-usd-rate-history`, 'POST', { fromDate, toDate, isAllData: false });
  },

  getGoldProducts: async () => {
    return proxyFetch(`${BASE_URL}/products/filter`, 'POST', {
      types: ['GOLD'],
      issuerIds: [],
      page: 1,
      pageSize: 100,
      fundAssetTypes: [],
      bondRemainPeriods: [],
      searchField: '',
    });
  },
};
