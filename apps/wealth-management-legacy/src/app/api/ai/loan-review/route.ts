import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import {
  extractAndParseJSON,
  STRUCTURED_INSIGHT_FORMAT_INSTRUCTION,
  type StructuredInsight,
} from '@wealth-management/ai/server';
import { Loan } from '@wealth-management/types';
import { AppError, LoanError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

interface EnrichedLoan extends Loan {
  type?: string;
  category?: string;
}

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as { loans: Loan[] };
    const { loans } = body;

    const totalDebt = loans.reduce((sum: number, loan: Loan) => sum + (loan.yearlyRemaining || 0), 0);

    const model = getLanguageModel('github-gpt-4.1');

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
    const actionPrompt = (await loadActionPrompt('loan-review')) + '\n\n' + STRUCTURED_INSIGHT_FORMAT_INSTRUCTION;

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: actionPrompt,
    });

    try {
      const structured = extractAndParseJSON<StructuredInsight>(text);
      return NextResponse.json({ review: structured });
    } catch {
      return NextResponse.json({ review: text });
    }
  } catch (error: unknown) {
    if (isAppError(error)) {
      console.error('AI Loan Review Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new LoanError(getErrorMessage(error), {
      userMessage: 'Unable to generate loan review. Please try again.',
    });
    console.error('AI Loan Review Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
