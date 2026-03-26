import { generateText } from 'ai';
import { getLanguageModel, type AIModelId } from '../providers';
import { buildSystemPrompt } from '../system-prompt';
import { extractAndParseJSON } from './parser';

interface OrchestratorOptions {
  modelId?: AIModelId;
  systemPromptInstruction?: string;
  prompt: string;
  maxTokens?: number;
  temperature?: number;
  experimental_json_mode?: boolean;
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
      maxTokens: options.maxTokens,
      // Pass through experimental_json_mode if explicitly requested
      // Note: Not all models support this, so we use it as a hint for the provider
      experimental_output: options.experimental_json_mode
        ? {
            type: 'object',
          }
        : undefined,
    } as any);

    return text;
  }

  /**
   * Executes a task and returns a type-safe JSON object.
   * Standardizes on asking for JSON in the prompt and then parsing it.
   */
  static async runJson<T>(options: OrchestratorOptions): Promise<T> {
    // Force prompt to emphasize JSON if not already present
    const enhancedOptions = {
      ...options,
      prompt: options.prompt.includes('JSON')
        ? options.prompt
        : `${options.prompt}\n\nIMPORTANT: Respond with a single valid JSON object strictly following the schema. No conversational preamble.`,
    };

    const text = await this.run(enhancedOptions);
    return extractAndParseJSON<T>(text);
  }
}
