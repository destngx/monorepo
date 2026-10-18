import { describe, expect, it } from 'vitest';
import { apiTokenProblem, generateApiToken } from './apiToken';

describe('generateApiToken', () => {
  it('makes a fresh 64 character hex token each time', () => {
    const first = generateApiToken();
    expect(first).toMatch(/^[0-9a-f]{64}$/);
    expect(generateApiToken()).not.toBe(first);
  });

  it('makes tokens the engine accepts', () => {
    expect(apiTokenProblem(generateApiToken())).toBe('');
  });
});

describe('apiTokenProblem', () => {
  it('follows the engine rule', () => {
    expect(apiTokenProblem('a'.repeat(32))).toBe('');
    expect(apiTokenProblem('a'.repeat(31))).toMatch(/at least 32/);
    expect(apiTokenProblem('a'.repeat(257))).toMatch(/at most 256/);
    expect(apiTokenProblem(`${'a'.repeat(32)} b`)).toMatch(/no spaces/);
    expect(apiTokenProblem(`${'a'.repeat(32)}é`)).toMatch(/visible ASCII/);
  });
});
