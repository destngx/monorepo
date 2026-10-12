import type { AIStatus, AIUseCase } from '$lib/api/types';

export type ModelSource = 'browser' | 'engine' | 'fallback' | 'none';

/**
 * The model a use case runs on (WM-SET-1): this browser's override, else the
 * engine's per-use-case model, else the engine's fallback model.
 */
export function effectiveModel(
  useCase: AIUseCase,
  status: Pick<AIStatus, 'models' | 'fallbackModel'> | null,
  override: string,
): { model: string; source: ModelSource } {
  if (override) return { model: override, source: 'browser' };
  const engine = status?.models[useCase];
  if (engine) return { model: engine, source: 'engine' };
  if (status?.fallbackModel) return { model: status.fallbackModel, source: 'fallback' };
  return { model: '', source: 'none' };
}
