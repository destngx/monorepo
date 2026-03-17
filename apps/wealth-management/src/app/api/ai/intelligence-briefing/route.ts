import { NextResponse } from 'next/server';
import { getLanguageModel } from "@wealth-management/ai/providers";
import { generateText } from 'ai';
import { buildSystemPrompt } from "@wealth-management/ai/server";

export async function POST(req: Request) {
  try {
    const { accounts, transactions, budget, loans, modelId } = await req.json();

    // Data Aggregation for AI
    const totalLoanDebt = loans?.reduce((s: number, l: any) => s + (l.yearlyRemaining || 0), 0) || 0;
    const totalAssets = accounts?.filter((a: any) => a.balance >= 0).reduce((s: number, a: any) => s + a.balance, 0) || 0;
    const totalLiabilities = (accounts?.filter((a: any) => a.balance < 0).reduce((s: number, a: any) => s + Math.abs(a.balance), 0) || 0) + totalLoanDebt;
    const netWorth = totalAssets - totalLiabilities;

    // Calculate Cash Flow (Current Month)
    const now = new Date();
    const currentMonthTxns = transactions?.filter((t: any) => {
      const d = new Date(t.date);
      return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
    }) || [];

    const income = currentMonthTxns.reduce((s: number, t: any) => s + (t.deposit || 0), 0);
    const expense = currentMonthTxns.reduce((s: number, t: any) => s + (t.payment || 0), 0);
    const cashFlow = income - expense;
    const savingsRate = income > 0 ? (cashFlow / income) * 100 : 0;

    const model = getLanguageModel(modelId || 'gpt-4o-mini');

    const taskInstruction = `
      Your task is to analyze the user's financial state and provide a "Daily Briefing" and a set of "Alerts & Nudges" to help them make better financial decisions.

      FINANCIAL SNAPSHOT:
      - Net Worth: ${netWorth} VND
      - Total Assets: ${totalAssets} VND
      - Total Liabilities: ${totalLiabilities} VND
      - Monthly Cash Flow: ${cashFlow} VND
      - Savings Rate: ${savingsRate.toFixed(1)}%
      
      RECENT DATA CONTEXT:
      - Accounts: ${JSON.stringify(accounts?.slice(0, 5))}... (and others)
      - Transactions: ${JSON.stringify(transactions?.slice(0, 10))}
      - Budget: ${JSON.stringify(budget?.slice(0, 5))}
      - Loans: ${JSON.stringify(loans)}

      REQUIREMENTS MUST RETURN:
      1. Briefing: A 2-3 sentence executive summary and Liquidity Signal. Be precise. Do not mention any numbers.
      2. Alerts & Nudges: Generate 2-4 items. Rank them by urgency ("critical", "warning", or "info"). What's my biggest financial risk right now?"
      3. Advanced Patterns:
         - **forecast**: Predict end-of-month budget status for primary categories.
         - **anomalies**: Identify spikes, duplicate charges, or unusual merchants.
         - **portfolioScore**: A score (0-100) with a 1-sentence breakdown (diversification, risk, liquidity).
         - **whatIf**: A single impactful "If... then..." scenario.
         - **benchmarking**: Compare savings rate or spending vs a hypothetical peer group in their income bracket.
         - **investments**: Tactical wealth-building advice (e.g., rebalancing, rotational strategies, opportunistic buys, or yield optimization based on the current market conditions).

      CRITICAL: All numbers in the JSON must be RAW NUMERIC values (e.g., 1250000), NOT formatted strings (e.g., "1.250.000" or 1.250.000). Do not use thousands separators.

      FORMAT: Return ONLY a STRICT JSON object:
      {
        "briefing": "string",
        "alerts": [
          { "type": "critical" | "warning" | "info", "title": "string", "message": "string" }
        ],
        "patterns": {
          "forecast": { "message": "string", "amount": number, "status": "on-track" | "at-risk" },
          "anomalies": [{ "date": "string", "title": "string", "amount": number, "reason": "string" }],
          "portfolioScore": { "score": number, "breakdown": "string" },
          "whatIf": { "scenario": "string", "impact": "string" },
          "benchmarking": { "message": "string", "percentile": number },
          "investments": { "title": "string", "tactic": "string", "opportunity": "string" }
        }
      }
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: "Generate the intelligence briefing and alerts based on the financial data provided. Ensure all JSON numbers are raw integers without separators.",
    });

    const match = text.match(/\{[\s\S]*\}/);
    if (!match) throw new Error("Invalid AI response");

    // Robulst cleaning: Replace numbers formatted with dots (e.g. 1.234.567) into pure numbers (1234567)
    // This targets values following a colon in the JSON string
    const cleanedJson = match[0].replace(/:\s*(\d[\d\.]+\d)/g, (m, p1) => {
      // If the number has multiple dots, it's definitely a formatted number (e.g. 2.916.834)
      // If it has one dot, it might be a decimal, but in this specific domain/VND, 
      // most large numbers with dots are thousands separators.
      // For VND, we usually don't have decimals. Let's be aggressive for now.
      return `: ${p1.replace(/\./g, '')}`;
    });

    const result = JSON.parse(cleanedJson);

    return NextResponse.json(result);
  } catch (error) {
    console.error('Intelligence Briefing Error:', error);
    return NextResponse.json({
      briefing: "Your financial dashboard is ready. Review your assets and liabilities to stay on track.",
      alerts: []
    }, { status: 500 });
  }
}
