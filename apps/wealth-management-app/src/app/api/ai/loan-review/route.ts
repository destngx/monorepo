import { NextResponse } from 'next/server';
import { getLanguageModel } from "@wealth-management/ai/providers";
import { generateText } from 'ai';
import { buildSystemPrompt } from "@wealth-management/ai/server";

export async function POST(req: Request) {
  try {
    const { loans } = await req.json();

    const totalDebt = loans.reduce((sum: number, loan: any) => sum + (loan.yearlyRemaining || 0), 0);

    const model = getLanguageModel('github-gpt-4o');

    const taskInstruction = `
      Your task is to provide a senior financial advisor's review of the user's loans and debts.
      Analyze the user's debt situation based on their loan information.
      
      Detailed Loans:
      ${JSON.stringify(loans.map((l: any) => ({
      name: l.name,
      remaining: l.yearlyRemaining,
      isLending: l.type === 'Lending',
      category: l.category
    })))}
      
      Total Debt: ${totalDebt} VND

      SPECIALIZED OBJECTIVES for LOANS PAGE:
      You are a strict, pragmatic Debt Elimination Strategist. Your only focus is structuring a mathematically or psychologically optimal way out of this exact debt profile.
      
      1. Snowball vs. Avalanche Recommendation: Explicitly recommend either the Debt Snowball (smallest balance first for psychological wins) or Debt Avalanche (highest interest first for mathematical savings) method specifically suited to their detailed loans.
      2. Debt-to-Income Reality Check: Speak directly on the burden of their total debt (${totalDebt} VND) relative to standard local benchmarks.
      3. Collection Strategy: If they have "Lending" items (money owed to them), provide tactical, professional advice on how to gracefully but firmly collect those debts.
      4. Single Next Action: Give exactly ONE urgent next step to execute this month.
      
      Format with structured, commanding bullet points. Do not give general budgeting advice; talk exclusively about debt structure and elimination pacing.
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: "Develop a debt settlement strategy based on the loan data provided.",
    });

    return NextResponse.json({ review: text });
  } catch (error) {
    console.error('AI Loan Review Error:', error);
    return NextResponse.json({ error: 'Failed to generate loan review' }, { status: 500 });
  }
}
