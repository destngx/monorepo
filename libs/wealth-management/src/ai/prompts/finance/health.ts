export interface HealthData {
  netWorth: number;
  totalAssets: number;
  totalLiabilities: number;
  loans: any[];
  months: any[];
}

export function buildFinancialHealthPrompt(data: HealthData): string {
  return `
Your task is to analyze the user's financial health with Vietnamese local expertise and strict VND formatting.

Current Position:
- Net Worth: ${data.netWorth} VND
- Total Assets: ${data.totalAssets} VND
- Total Liabilities: ${data.totalLiabilities} VND
- Detailed External Debts: ${JSON.stringify(data.loans)}

Recent Performance (Last 3 Months):
${JSON.stringify(data.months)}

SPECIALIZED OBJECTIVES for DASHBOARD EXECUTIVE SUMMARY:
You are the ultimate CFO. Do not get bogged down in micro-transactions. Your job is to deliver a holistic, top-level executive summary of their momentum.

EXPECTED OUTPUT:
1. Provide a "Financial Health Score" from 0 to 100 based on modern wealth-building principles (e.g., 50/30/20 adherence, debt-to-asset ratio, liquidity).
2. Provide exactly 3 "Signals" covering the Financial Trinity:
   - **Liquidity Signal**: Are they cash-rich or cash-poor?
   - **Solvency Signal**: Is their debt-to-asset safe or toxic?
   - **Profitability Signal**: Based on recent months, are they actually saving money (Net Worth growing) or bleeding?

For each signal, you must evaluate:
   - type: "good" | "warn" | "bad"
   - title: Short bold title
   - detail: One sharp, executive-level sentence explanation

Format your response as a STRICT JSON object:
{
  "score": number,
  "signals": [
    { "type": "good" | "warn" | "bad", "title": string, "detail": string },
    ...
  ]
}
  `.trim();
}
