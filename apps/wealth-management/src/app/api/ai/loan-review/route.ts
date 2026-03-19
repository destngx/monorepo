import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import { Loan } from '@wealth-management/types';

interface EnrichedLoan extends Loan {
  type?: string;
  category?: string;
}

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as { loans: Loan[] };
    const { loans } = body;

    const totalDebt = loans.reduce((sum: number, loan: Loan) => sum + (loan.yearlyRemaining || 0), 0);

    const model = getLanguageModel('github-gpt-4o');

    const taskTemplate = await loadTaskPrompt('loan-review');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      totalDebt,
      loansJson: JSON.stringify(
        loans.map((l: EnrichedLoan) => ({
          name: l.name,
          remaining: l.yearlyRemaining,
          isLending: l.type === 'Lending',
          category: l.category,
        })),
      ),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionPrompt = await loadActionPrompt('loan-review');

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: actionPrompt,
    });

    return NextResponse.json({ review: text });
  } catch (error: unknown) {
    console.error('AI Loan Review Error:', error);
    return NextResponse.json({ error: 'Failed to generate loan review' }, { status: 500 });
  }
}
