import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({
  startGoogleAuth: vi.fn(),
  completeGoogleAuth: vi.fn(),
}));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
const signIn = vi.hoisted(() => ({ signInHere: vi.fn() }));
vi.mock('./googleAuth', () => signIn);

import GoogleConnect from './GoogleConnect.svelte';

const start = {
  authUrl: 'https://accounts.google.com/o/oauth2/auth?state=s1',
  redirectUrl: 'http://127.0.0.1:8080/api/auth/google/callback',
  purpose: 'sheets' as const,
};

async function settle() {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
}

describe('GoogleConnect', () => {
  let onchange: ReturnType<typeof vi.fn<() => Promise<void>>>;

  beforeEach(() => {
    vi.clearAllMocks();
    // jsdom has no modal dialogs; the Modal only needs them to toggle.
    if (!('showModal' in HTMLDialogElement.prototype)) {
      Object.assign(HTMLDialogElement.prototype, {
        showModal(this: HTMLDialogElement) {
          this.open = true;
        },
        close(this: HTMLDialogElement) {
          this.open = false;
        },
      });
    }
    signIn.signInHere.mockReturnValue(false);
    onchange = vi.fn(() => Promise.resolve());
    endpoints.startGoogleAuth.mockResolvedValue(start);
  });

  const renderConnect = (disabled = false) =>
    render(GoogleConnect, { purpose: 'sheets', label: 'Connect Google account', disabled, onchange });

  it('sends this tab to Google when Google returns to this page', async () => {
    signIn.signInHere.mockReturnValue(true);
    renderConnect();

    await fireEvent.click(screen.getByRole('button', { name: 'Connect Google account' }));
    await settle();

    expect(endpoints.startGoogleAuth).toHaveBeenCalledWith('sheets');
    expect(signIn.signInHere).toHaveBeenCalledWith(start, window.location);
    expect(screen.getByRole('button', { name: 'Opening Google...' })).toBeTruthy();
    expect(screen.queryByText(start.redirectUrl)).toBeNull();
  });

  it('offers a new tab and the paste fallback when Google returns elsewhere', async () => {
    renderConnect();

    await fireEvent.click(screen.getByRole('button', { name: 'Connect Google account' }));
    await settle();

    expect(screen.getByText(start.redirectUrl)).toBeTruthy();
    expect(screen.getByRole<HTMLAnchorElement>('link', { name: 'Open Google sign-in' }).href).toBe(start.authUrl);
  });

  it('reports the error when the engine refuses to start', async () => {
    const failure = new Error('set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in Settings and save them first');
    endpoints.startGoogleAuth.mockRejectedValue(failure);
    renderConnect();

    await fireEvent.click(screen.getByRole('button', { name: 'Connect Google account' }));
    await settle();

    expect(signIn.signInHere).not.toHaveBeenCalled();
    expect(toastMock.error).toHaveBeenCalledWith(failure, 'Connect Google account');
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Connect Google account' }).disabled).toBe(false);
  });

  it('completes the sign-in from a pasted address and reloads the settings', async () => {
    endpoints.completeGoogleAuth.mockResolvedValue({
      purpose: 'sheets',
      settingKey: 'GOOGLE_REFRESH_TOKEN',
      restartRequired: true,
    });
    renderConnect();
    await fireEvent.click(screen.getByRole('button', { name: 'Connect Google account' }));
    await settle();

    const pasted = `${start.redirectUrl}?state=s1&code=4%2Fabc`;
    await fireEvent.input(screen.getByLabelText(/paste its address/), { target: { value: pasted } });
    await fireEvent.click(screen.getByRole('button', { name: 'Complete' }));
    await settle();

    expect(endpoints.completeGoogleAuth).toHaveBeenCalledWith(pasted);
    expect(toastMock.push).toHaveBeenCalledWith('success', expect.stringContaining('GOOGLE_REFRESH_TOKEN'));
    expect(onchange).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(start.redirectUrl)).toBeNull();
  });

  it('keeps the dialog open with the error when the pasted address is rejected', async () => {
    endpoints.completeGoogleAuth.mockRejectedValue(new Error('this Google sign-in is unknown or expired'));
    renderConnect();
    await fireEvent.click(screen.getByRole('button', { name: 'Connect Google account' }));
    await settle();

    await fireEvent.input(screen.getByLabelText(/paste its address/), {
      target: { value: 'http://x/?state=a&code=b' },
    });
    await fireEvent.keyDown(screen.getByLabelText(/paste its address/), { key: 'Enter' });
    await settle();

    expect(screen.getByRole('alert').textContent).toContain('unknown or expired');
    expect(onchange).not.toHaveBeenCalled();
  });

  it('reloads the settings when the user comes back from the Google tab or presses Done', async () => {
    renderConnect();
    await fireEvent.click(screen.getByRole('button', { name: 'Connect Google account' }));
    await settle();

    await fireEvent.focus(window);
    expect(onchange).toHaveBeenCalledTimes(1);

    await fireEvent.click(screen.getByRole('button', { name: 'Done' }));
    await settle();
    expect(onchange).toHaveBeenCalledTimes(2);
    expect(screen.queryByText(start.redirectUrl)).toBeNull();
  });

  it('cannot start while the button is disabled', () => {
    renderConnect(true);

    const button = screen.getByRole<HTMLButtonElement>('button', { name: 'Connect Google account' });
    expect(button.disabled).toBe(true);
  });
});
