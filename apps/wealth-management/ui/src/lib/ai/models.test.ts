import { describe, expect, it } from 'vitest';
import { effectiveModel } from './models';

const status = { models: { insight: 'gpt-insight' }, fallbackModel: 'gpt-fallback' };

describe('effectiveModel', () => {
  it('prefers the browser override', () => {
    expect(effectiveModel('insight', status, 'mine')).toEqual({ model: 'mine', source: 'browser' });
  });

  it('then the engine model for the use case, then the fallback', () => {
    expect(effectiveModel('insight', status, '')).toEqual({ model: 'gpt-insight', source: 'engine' });
    expect(effectiveModel('chat', status, '')).toEqual({ model: 'gpt-fallback', source: 'fallback' });
  });

  it('reports nothing when the status is unknown', () => {
    expect(effectiveModel('chat', null, '')).toEqual({ model: '', source: 'none' });
  });
});
