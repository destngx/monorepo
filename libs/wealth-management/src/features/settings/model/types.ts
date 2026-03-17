export interface Settings {
  provider: 'openai' | 'google' | 'anthropic';
  model: string;
  currency: 'VND' | 'USD';
}
