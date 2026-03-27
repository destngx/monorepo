'use server';

import { invalidateCache } from '@wealth-management/utils';
import { revalidatePath } from 'next/cache';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';
import { env } from '@wealth-management/config';

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
    github: !!env.ai.githubToken,
    openai: !!env.ai.openaiApiKey,
    anthropic: !!env.ai.anthropicApiKey,
    google: !!env.ai.googleGenAiApiKey,
  };
}
