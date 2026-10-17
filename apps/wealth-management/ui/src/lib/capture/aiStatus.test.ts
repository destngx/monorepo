import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '$lib/api/client';
import type { AIStatus } from '$lib/api/types';

const aiStatus = vi.hoisted(() => vi.fn());
vi.mock('$lib/api/endpoints', () => ({ api: { aiStatus } }));

import { captureAIStatus } from './aiStatus';

describe('captureAIStatus', () => {
  it('Given the engine answers When asked Then the status is returned', async () => {
    const status = { configured: true } as AIStatus;
    aiStatus.mockResolvedValueOnce(status);
    await expect(captureAIStatus()).resolves.toBe(status);
  });

  it('Given the engine has no AI routes (404) When asked Then it is null, meaning AI is off', async () => {
    aiStatus.mockRejectedValueOnce(new ApiError(404, 'not found'));
    await expect(captureAIStatus()).resolves.toBeNull();
  });

  it('Given another failure When asked Then it is not hidden', async () => {
    aiStatus.mockRejectedValueOnce(new ApiError(500, 'engine broke'));
    await expect(captureAIStatus()).rejects.toThrow('engine broke');
  });
});
