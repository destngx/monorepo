"use server";

import { invalidateCache } from "@wealth-management/utils";
import { revalidatePath } from "next/cache";

export async function clearAllCache() {
  try {
    // Invalidate our internal Prisma cache explicitly
    await invalidateCache('accounts');
    await invalidateCache('transactions');
    await invalidateCache('budget');

    // Invalidate Next.js server cache for all routes
    revalidatePath('/', 'layout');

    return { success: true };
  } catch (error) {
    console.error("Failed to clear cache:", error);
    return { success: false, error: "Failed to clear cache" };
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
