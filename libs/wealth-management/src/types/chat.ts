/**
 * Chat domain types
 */

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt?: Date;
}

export interface ChatResponse {
  message: ChatMessage;
  error?: string;
}

export interface SuggestionItem {
  label: string;
  prompt: string;
}

export interface SuggestionsResponse {
  suggestions: SuggestionItem[];
}

export interface AIInsight {
  id: string;
  title: string;
  content: string;
  createdAt: Date;
  metadata?: Record<string, any>;
}

export interface AIModelConfig {
  provider: 'openai' | 'anthropic' | 'google' | 'github';
  model: string;
  label: string;
  description: string;
}

export const AI_MODELS: Record<string, AIModelConfig> = {
  'gpt-4o': { provider: 'openai', model: 'gpt-4o', label: 'GPT-4o (OpenAI)', description: 'Best quality, higher cost' },
  'gpt-4o-mini': {
    provider: 'openai',
    model: 'gpt-4o-mini',
    label: 'GPT-4o Mini (OpenAI)',
    description: 'Good quality, lower cost',
  },
  'claude-3-5-sonnet-20240620': {
    provider: 'anthropic',
    model: 'claude-3-5-sonnet-20240620',
    label: 'Claude Sonnet',
    description: 'Balanced quality and speed',
  },
  'github-gpt-4.1': {
    provider: 'github',
    model: 'gpt-4.1',
    label: 'GPT-4.1 (GitHub)',
    description: 'Next-gen advanced reasoning',
  },
  'github-gpt-4o': {
    provider: 'github',
    model: 'gpt-4o',
    label: 'GPT-4o (GitHub)',
    description: 'Standard high-reasoning model',
  },
  'github-grok': {
    provider: 'github',
    model: 'grok-code-fast-1',
    label: 'Grok (GitHub)',
    description: 'Fast perspective model',
  },
  'gemini-3.1-flash-lite': {
    provider: 'github',
    model: 'gemini-3-flash-preview',
    label: 'Gemini 3.1 Flash-Lite',
    description: "Google's most cost-efficient Gemini model",
  },
  'github-gemini-3.1-pro': {
    provider: 'github',
    model: 'gemini-3-pro-preview',
    label: 'Gemini 3.1 Pro (GitHub)',
    description: 'Modern reasoning via Copilot',
  },
  'github-claude-haiku-4.5': {
    provider: 'github',
    model: 'claude-haiku-4.5',
    label: 'Claude Haiku 4.5 (GitHub)',
    description: 'Fast and intelligent small model',
  },
};

export type AIModelId = keyof typeof AI_MODELS;
