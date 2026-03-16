import { describe, it, expect, vi, beforeEach } from 'vitest';
import { getLanguageModel } from './providers';
import { openai } from '@ai-sdk/openai';
import { google } from '@ai-sdk/google';
import { anthropic } from '@ai-sdk/anthropic';

describe('AI Providers', () => {
  it('returns openai model as default', () => {
    const model = getLanguageModel('gpt-4o-mini');
    expect(model).toBeDefined();
    // Assuming factory function return
  });

  it('handles google gemini models', () => {
    const model = getLanguageModel('gemini-2.5-flash');
    expect(model).toBeDefined();
  });

  it('handles anthropic claude models', () => {
    const model = getLanguageModel('claude-3-5-sonnet');
    expect(model).toBeDefined();
  });
});
