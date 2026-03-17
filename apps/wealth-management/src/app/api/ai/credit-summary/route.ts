import { NextResponse } from 'next/server';
import { getLanguageModel } from "@wealth-management/ai/providers";
import { generateText } from 'ai';
import { buildSystemPrompt } from "@wealth-management/ai/server";
import { Transaction } from '@wealth-management/types';

export async function POST(req: Request) {
  try {
    const body = await req.json() as {
      transactions: Transaction[];
      cardStats: unknown;
      targetMonth: string;
    };
    const { transactions, cardStats, targetMonth } = body;

    // targetMonth is "MM/YYYY" or "current"
    const isPastMonth = targetMonth && targetMonth !== 'current';
    const monthLabel = isPastMonth ? `for the month of ${targetMonth}` : 'for the current month';

    // Filter transactions to the selected month if provided
    let filteredTransactions = transactions;
    if (isPastMonth) {
      const [m, y] = targetMonth.split('/');
      filteredTransactions = transactions.filter((t: Transaction) => {
        const d = new Date(t.date);
        return d.getMonth() + 1 === parseInt(m) && d.getFullYear() === parseInt(y);
      });
    }

    const recentTransactions = filteredTransactions
      .slice(-100)
      .map((t: Transaction) => ({
        payee: t.payee,
        category: t.category,
        amount: t.payment,
        tags: t.tags,
        date: t.date
      }));

    const model = getLanguageModel('github-gpt-4o');

    const taskInstruction = `
      Your task is to provide senior-level optimization for the user's Sacombank credit cards ${monthLabel}.
      
      ANALYSIS FOCUS:
      Current analysis target is ${monthLabel}. Ensure your findings and suggestions are specific to the behavior observed in this dataset.
      
      SACCOMBANK CREDIT PRODUCTS:
      1. Sacombank Visa UNIQ Platinum:
         - 20% cashback on Supermarket & Transport (MCC based, tags like "uniq supermarket and transport").
         - *CRITICAL MCC RULE FOR TRANSPORT:* "Transportation" means ride-hailing (Grab, Be, Gojek), taxis, and similar mobility services. **Gas stations and fuel purchases do NOT count as transportation.** Gas stations fall under a different MCC and only yield the standard 0.5% cashback.
         - 0.5% on others (tags like "uniq others").
         - Cap: 300,000 VND for high-rate categories, 600,000 VND total per cycle.
      
      2. Sacombank Visa Platinum Cashback:
         - 5% cashback on Online transactions (tags like "platinum online").
         - 3% cashback on Overseas POS.
         - 0.5% on Airline/others (tags like "platinum").
         - Cap: 600,000 VND total per cycle.
       
       *IMPORTANT USAGE NOTES:* 
       - **Minimum Threshold:** For Sacombank Visa UNIQ Platinum, a minimum cashback amount of **10,000 VND** per billing cycle is required for the refund to be credited; otherwise, it is forfeited for that cycle.
       - **Tagging Context:** The tags provided in the transaction data are manuals applied by the user as a **reference for which card was used**. They do NOT guarantee that the transaction actually follows the cashback rules. Treat tags as a reference of *intent*, but verify if the merchant or category alignment makes sense for the high-reward rates.
       - **Statement Delay:** Sacombank cashback refunds typically appear as card statement credits after a **1-3 month delay**.

      USER DATA:
      Current Stats: ${JSON.stringify(cardStats)}
      Recent Transactions: ${JSON.stringify(recentTransactions)}
      Shared Limit: 40M VND total.

      CONSTRAINTS & SPECIALIZED OBJECTIVES:
      You are a strict, hyper-optimized Credit Reward Engine. Your only goal is to maximize the user's cashback and highlight their mistakes.
      
      *PRIMARY GROUNDING RULE:* All performance reviews, audits, and optimization suggestions MUST be grounded in the **Actual Cashback** (refunds) received from the dataset. While estimated cashback shows potential, the only metric that matters for performance evaluation is real money returned to the card.

      EFFICIENCY METRICS:
      - **Actual Efficiency:** Calculated as (Actual Refund / Spent) * 100. This represents the REAL money back in the pocket. This is your primary metric.
      - **Estimated Efficiency:** Calculated as (Estimated Cashback / Spent) * 100. This is the potential if everything goes perfectly.
      - **The Gap:** Always comment on the gap between Estimated and Actual efficiency. If Actual is significantly lower, diagnose why (e.g., forgotten refunds, ineligible transactions, or the 1-3 month statement delay).

      1. Efficiency Deep-Dive: Analyze the user's efficiency metrics from the provided data. Since Actual Cashback is your grounding truth, if current month efficiency is low, tell them why.
      2. Tag vs. Reality: If you see a transaction with a "high-reward" tag but the payee or category looks suspicious (e.g., fuel tagged as "transport" for UNIQ), point out that this tag is just a reference and may not actually yield the expected 20% cashback.
      3. Cash Leakage Identification: Point out specific transactions where they used the wrong card. Calculate the exact VND they missed out on.
      4. Cap & Minimum Warning: State clearly how close they are to the 600k VND monthly cap AND if they are at risk of missing the 10,000 VND minimum threshold for UNIQ. **CRITICAL:** Always use the ACTUAL Cashback (refunds) value to justify how much of the cap has truly been utilized. Avoid relying on estimates for cap awareness if actual data is available.
      5. Upcoming Action: Tell them exactly which card to pull out for their next physical and online purchase.
      
      Format with aggressive, high-value bullet points. Do not give general budgeting advice; talk exclusively about card optimization and points/cashback math.
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: "Generate a credit card optimization summary.",
    });

    return NextResponse.json({ summary: text });
  } catch (error: unknown) {
    console.error('AI Summary API Error:', error);
    const message = error instanceof Error ? error.message : 'Failed to generate summary';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
