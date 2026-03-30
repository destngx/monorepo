const API_URL = '/api/fmarket';

const serverFetch = async (action: string, params: any = {}) => {
  const response = await fetch(API_URL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      action,
      params,
    }),
  });
  return response.json();
};

export const fmarketApi = {
  getProductsFilterNav: async (page = 1, pageSize = 10, assetTypes = ['STOCK'], isMMFFund = false) => {
    return serverFetch('getProductsFilterNav', { page, pageSize, assetTypes, isMMFFund });
  },

  getIssuers: async () => {
    return serverFetch('getIssuers');
  },

  getBankInterestRates: async () => {
    return serverFetch('getBankInterestRates');
  },

  getProductDetails: async (code: string) => {
    return serverFetch('getProductDetails', { code });
  },

  getNavHistory: async (productId: number, navPeriod = 'navToBeginning') => {
    return serverFetch('getNavHistory', { productId, navPeriod });
  },

  getGoldPriceHistory: async (fromDate: string, toDate: string) => {
    return serverFetch('getGoldPriceHistory', { fromDate, toDate });
  },

  getUsdRateHistory: async (fromDate: string, toDate: string) => {
    return serverFetch('getUsdRateHistory', { fromDate, toDate });
  },

  getGoldProducts: async () => {
    return serverFetch('getGoldProducts');
  },
};
