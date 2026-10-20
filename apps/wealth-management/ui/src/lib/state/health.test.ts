import { beforeEach, describe, expect, it } from 'vitest';
import type { DependencyStatus, EngineHealth } from '$lib/api/types';
import { engineHealth } from './health.svelte';
import { toasts } from './toasts.svelte';

const expired = { message: 'oauth2: "invalid_grant" "Token has been expired or revoked."', kind: 'expired' as const };

function health(sheets: DependencyStatus): EngineHealth {
  return { status: sheets, checks: [{ name: 'sheets', status: sheets }] };
}

describe('engineHealth.report', () => {
  beforeEach(() => {
    engineHealth.settled = null;
    toasts.sheetsAuthProblem = expired;
  });

  it('keeps the answer', () => {
    engineHealth.report({ ok: true, value: health('ok') });
    expect(engineHealth.ready('sheets')).toBe(true);
  });

  it('clears the re-auth alert once the sheets check passes again', () => {
    engineHealth.report({ ok: true, value: health('ok') });
    expect(toasts.sheetsAuthProblem).toBeNull();
  });

  it('keeps the re-auth alert while the sheets check still fails', () => {
    engineHealth.report({ ok: true, value: health('down') });
    expect(toasts.sheetsAuthProblem).toEqual(expired);
  });

  it('keeps the re-auth alert when the engine does not answer', () => {
    engineHealth.report({ ok: false, error: 'Engine unreachable' });
    expect(toasts.sheetsAuthProblem).toEqual(expired);
  });

  it('keeps the re-auth alert while the first check is in flight', () => {
    engineHealth.report(null);
    expect(toasts.sheetsAuthProblem).toEqual(expired);
  });
});
