import type { GoogleAuthStart } from '$lib/api/types';
import type { ToastKind } from '$lib/state/toasts.svelte';

/** The query the engine's Google callback sends the browser back with. */
export const googleAuthParams = ['google', 'purpose', 'reason'] as const;

const reasons: Record<string, string> = {
  denied: 'Google sign-in was cancelled or refused.',
  expired: 'This Google sign-in expired or was already used. Connect again.',
  rejected: 'Google rejected the sign-in code. Connect again.',
  no_refresh_token:
    "Google returned no refresh token. Remove the app's access in your Google account (Security > Third-party connections), then connect again.",
  invalid: 'The Google sign-in response was incomplete. Connect again.',
};

/**
 * What to tell the user after the engine's Google callback redirected here,
 * or null when the page was opened normally. Reasons are fixed codes, so a
 * crafted link cannot show arbitrary text.
 */
export function googleAuthNotice(params: URLSearchParams): { kind: ToastKind; message: string } | null {
  switch (params.get('google')) {
    case 'connected': {
      const account = params.get('purpose') === 'test' ? 'Test account' : 'Google account';
      return { kind: 'success', message: `${account} connected. Restart the engine to use it.` };
    }
    case 'error':
      return {
        kind: 'error',
        message: reasons[params.get('reason') ?? ''] ?? 'Saving the Google sign-in failed; see the engine log.',
      };
    default:
      return null;
  }
}

/**
 * Whether Google returns the browser to this page's origin. Then the sign-in
 * runs in this tab and the callback brings it back to Settings; otherwise
 * (GOOGLE_REDIRECT_URL elsewhere, or the UI opened from another host) the
 * return address may not be reachable from here and the user pastes it.
 */
export function returnsHere(start: GoogleAuthStart, origin: string): boolean {
  try {
    return new URL(start.redirectUrl).origin === origin;
  } catch {
    return false;
  }
}

/**
 * Sends this tab to Google when Google returns here; false when the sign-in
 * has to run in another tab with the paste fallback.
 */
export function signInHere(start: GoogleAuthStart, here: Pick<Location, 'origin' | 'assign'>): boolean {
  if (!returnsHere(start, here.origin)) return false;
  here.assign(start.authUrl);
  return true;
}
