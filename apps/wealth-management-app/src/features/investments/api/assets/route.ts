import { NextResponse } from 'next/server';
import { readSheet } from "@wealth-management/services/server";
import { handleApiError } from "@wealth-management/utils/server";

export async function GET() {
  try {
    const [cryptoData, fundData] = await Promise.all([
      readSheet('Crypto!A1:Z100'),
      readSheet('InvestmentFundCertificate!A1:Z100')
    ]);

    // Robust parser function
    const parseHoldings = (data: string[][], keywords: string[]) => {
      const headerIdx = data.findIndex(row => row.some(cell => keywords.includes(cell.trim())));
      if (headerIdx === -1) return { headers: [], holdings: [] };

      const headersRaw = data[headerIdx];
      const headers = headersRaw.map((h, i) => i === 0 && !h ? 'Platform' : (h || `Col${i}`));
      let lastPlatform = '';

      const holdings = data.slice(headerIdx + 1).map(row => {
        const item: Record<string, any> = {};
        if (row[0]) lastPlatform = row[0];

        headers.forEach((header, index) => {
          if (index === 0) {
            item[header] = lastPlatform;
          } else {
            item[header] = row[index] || '';
          }
        });
        return item;
      }).filter(item => {
        // Keep if it has non-platform data
        const otherKeys = headers.filter(h => h !== 'Platform');
        return otherKeys.some(k => item[k] !== '');
      });

      return { headers, holdings };
    };

    const crypto = parseHoldings(cryptoData, ['Currency', 'Spot/Fund', 'Token']);
    const funds = parseHoldings(fundData, ['Index', 'Total Unit', 'Certificate']);

    return NextResponse.json({
      crypto,
      funds
    });


  } catch (error) {
    return handleApiError(error, 'Investment Assets');
  }
}
