import { describe, expect, it } from 'vitest';
import { isPublicPath, safeNext } from './paths';

describe('isPublicPath', () => {
  it('matches the auth pages and their children only', () => {
    expect(isPublicPath('/login')).toBe(true);
    expect(isPublicPath('/invite/hwi_abc')).toBe(true);
    expect(isPublicPath('/capture')).toBe(true);
    expect(isPublicPath('/loginx')).toBe(false);
    expect(isPublicPath('/')).toBe(false);
  });
});

describe('safeNext', () => {
  it('keeps app paths and drops other origins', () => {
    expect(safeNext('/people/1?x=1')).toBe('/people/1?x=1');
    expect(safeNext('//evil.example')).toBe('/');
    expect(safeNext('https://evil.example')).toBe('/');
    expect(safeNext(null)).toBe('/');
  });
});
