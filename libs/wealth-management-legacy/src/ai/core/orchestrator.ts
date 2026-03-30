import { generateText, type LanguageModel } from 'ai';
import { getLanguageModel, type AIModelId } from '../providers';
import { buildSystemPrompt } from '../system-prompt';
import { extractAndParseJSON } from './parser';

interface OrchestratorOptions {
  modelId?: AIModelId;
  systemPromptInstruction?: string;
  prompt: string;
  maxTokens?: number;
  temperature?: number;
}

/**
 * AI Orchestrator: "The Divine Engine"
 *
 * A unified service to handle AI model selection, dynamic system prompt construction,
 * execution, and response parsing.
 */
export class AIOrchestrator {
  /**
   * Executes a standard text generation task.
   */
  static async run(options: OrchestratorOptions): Promise<string> {
    const model = getLanguageModel(options.modelId || 'github-gpt-4o');
    const system = await buildSystemPrompt(options.systemPromptInstruction);

    const { text } = await generateText({
      model,
      system,
      prompt: options.prompt,
      temperature: options.temperature,
    });

    return text;
  }

  /**
   * Executes a task and returns a type-safe JSON object.
   */
  static async runJson<T>(options: OrchestratorOptions): Promise<T> {
    const text = await this.run(options);
    return extractAndParseJSON<T>(text);
  }
}
