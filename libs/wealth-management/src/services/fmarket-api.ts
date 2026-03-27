export const fmarketApi = {
  getProductDetails: async (symbol: string): Promise<any> => {
    return { status: 404, data: null };
  },
  getTickerDetails: async (symbol: string): Promise<any> => {
    return { status: 404, data: null };
  },
  getNavHistory: async (fundId: string | number): Promise<any> => {
    return { status: 404, data: null };
  },
};
