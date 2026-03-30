'use server';

import { invalidateCache } from '@wealth-management/utils';
import { revalidatePath } from 'next/cache';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function clearAllCache() {
  try {
    // Invalidate our internal Prisma cache explicitly
    await invalidateCache('accounts');
    await invalidateCache('transactions');
    await invalidateCache('budget');

    // Invalidate Next.js server cache for all routes
    revalidatePath('/', 'layout');

    return { success: true };
  } catch (error: unknown) {
    if (isAppError(error)) {
      return { success: false, error: error.userMessage };
    }
    const message = getErrorMessage(error);
    const appError = new AppError('Failed to clear cache', undefined, 500, undefined, {
      context: { originalError: message },
      userMessage: 'Cache clearing failed. Please try again.',
    });
    return { success: false, error: appError.userMessage };
  }
}

export async function getCredentialStatuses() {
  return {
    github: !!process.env.GITHUB_TOKEN,
    openai: !!process.env.OPENAI_API_KEY,
    anthropic: !!process.env.ANTHROPIC_API_KEY,
    google: !!process.env.GOOGLE_GENERATIVE_AI_API_KEY,
  };
}
