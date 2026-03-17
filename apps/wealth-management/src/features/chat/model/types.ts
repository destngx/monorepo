/**
 * Chat domain types
 */

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
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
