import { describe, expect, it, vi } from 'vitest';
import { googleAuthNotice, returnsHere, signInHere } from './googleAuth';

const start = (redirectUrl: string) => ({
  authUrl: 'https://accounts.google.com/x',
  redirectUrl,
  purpose: 'sheets' as const,
});

describe('googleAuthNotice', () => {
  it('confirms a connected account and asks for a restart', () => {
    expect(googleAuthNotice(new URLSearchParams('google=connected&purpose=sheets'))).toEqual({
      kind: 'success',
      message: 'Google account connected. Restart the engine to use it.',
    });
  });

  it('explains each failure reason with fixed text', () => {
    expect(googleAuthNotice(new URLSearchParams('google=error&reason=expired'))).toEqual({
      kind: 'error',
      message: 'This Google sign-in expired or was already used. Connect again.',
    });
    expect(googleAuthNotice(new URLSearchParams('google=error&reason=no_refresh_token'))?.message).toMatch(
      /Third-party connections/,
    );
  });

  it('never shows text from the link itself', () => {
    const notice = googleAuthNotice(new URLSearchParams('google=error&reason=Call+0900+for+help'));
    expect(notice?.message).toBe('Saving the Google sign-in failed; see the engine log.');
  });

  it('says nothing on a normal visit', () => {
    expect(googleAuthNotice(new URLSearchParams(''))).toBeNull();
    expect(googleAuthNotice(new URLSearchParams('google=other'))).toBeNull();
  });
});

describe('returnsHere', () => {
  it('is true only when Google returns to this origin', () => {
    expect(returnsHere(start('http://127.0.0.1:5173/api/auth/google/callback'), 'http://127.0.0.1:5173')).toBe(true);
    expect(returnsHere(start('http://127.0.0.1:8080/api/auth/google/callback'), 'http://ezmacmini:5173')).toBe(false);
    expect(returnsHere(start('not a url'), 'http://127.0.0.1:5173')).toBe(false);
  });
});

describe('signInHere', () => {
  it('sends this tab to Google when Google returns here', () => {
    const here = { origin: 'http://127.0.0.1:5173', assign: vi.fn() };
    expect(signInHere(start('http://127.0.0.1:5173/api/auth/google/callback'), here)).toBe(true);
    expect(here.assign).toHaveBeenCalledWith('https://accounts.google.com/x');
  });

  it('stays when Google returns to an address this browser may not reach', () => {
    const here = { origin: 'http://ezmacmini:5173', assign: vi.fn() };
    expect(signInHere(start('http://127.0.0.1:8080/api/auth/google/callback'), here)).toBe(false);
    expect(here.assign).not.toHaveBeenCalled();
  });
});
