import { describe, expect, it } from 'vitest';
import { network } from './network.svelte';

describe('network', () => {
  it('reports only changes of connectivity', () => {
    expect(network.online).toBe(true);
    expect(network.update(true)).toBeNull();
    expect(network.update(false)).toBe('lost');
    expect(network.online).toBe(false);
    expect(network.update(false)).toBeNull();
    expect(network.update(true)).toBe('restored');
    expect(network.online).toBe(true);
  });
});
