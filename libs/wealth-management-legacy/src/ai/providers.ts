import { openai } from '@ai-sdk/openai';
import { createOpenAICompatible } from '@ai-sdk/openai-compatible';
import { google } from '@ai-sdk/google';
import { anthropic } from '@ai-sdk/anthropic';
import { AI_MODELS, AIModelConfig } from '@wealth-management/types/chat';
export type { AIModelId } from '@wealth-management/types/chat';

let cachedCopilotToken: { token: string; expiresAt: number } | null = null;

async function getCopilotToken() {
  const ghToken = process.env.GITHUB_TOKEN;
  if (!ghToken) throw new Error('GITHUB_TOKEN not set');

  if (cachedCopilotToken && cachedCopilotToken.expiresAt > Date.now() / 1000 + 300) {
    return cachedCopilotToken.token;
  }

  const res = await fetch('https://api.github.com/copilot_internal/v2/token', {
    headers: {
      Authorization: `token ${ghToken}`,
      Accept: 'application/json',
    },
    // Prevent Next.js from aggressively caching this token fetch
    cache: 'no-store',
  });

  if (!res.ok) throw new Error(`Failed to get Copilot token: ${res.status}`);
  const data = await res.json();
  cachedCopilotToken = { token: data.token, expiresAt: data.expires_at };
  return data.token;
}

// Set up GitHub Copilot using OpenAI SDK compatibility and a custom fetcher
const github = createOpenAICompatible({
  name: 'github',
  baseURL: 'https://api.githubcopilot.com',
  apiKey: 'dummy-key',
  fetch: async (url, init) => {
    // Override the URL to point to standard chat completions
    let fetchUrl = url.toString();
    if (fetchUrl.includes('/responses')) {
      fetchUrl = fetchUrl.replace('/responses', '/chat/completions');
    }

    if (init?.body) {
      try {
        const body = JSON.parse(init.body as string);

        // 1. Remove keys with undefined/null values (GitHub API is picky)
        Object.keys(body).forEach((key) => {
          if (body[key] === undefined || body[key] === null) {
            delete body[key];
          }
        });

        // 2. Fix Tool Schema types (SDK 6 sometimes misses 'type: object')
        if (body.tools) {
          body.tools.forEach((t: any) => {
            if (t.function?.parameters) {
              if (!t.function.parameters.type || t.function.parameters.type === 'None') {
                t.function.parameters.type = 'object';
              }
            }
          });
        }

        init.body = JSON.stringify(body);
      } catch (e) {
        console.error('[GitHub Provider Interceptor Error]:', e);
      }
    }

    const token = await getCopilotToken();
    const headers = new Headers(init?.headers);
    headers.set('Authorization', `Bearer ${token}`);
    headers.set('Editor-Version', 'vscode/1.80.0');
    headers.set('Editor-Plugin-Version', 'copilot-chat/0.1.0');

    return fetch(fetchUrl, { ...init, headers });
  },
});

export function getLanguageModel(modelId: string) {
  const config = AI_MODELS[modelId];

  // If modelId is not recognized, attempt to choose a default based on available keys
  if (!config) {
    if (modelId.startsWith('github-')) return github(modelId.replace('github-', ''));
    return github('gpt-4o');
  }

  switch (config.provider) {
    case 'openai':
      return openai(config.model);
    case 'google':
      return google(config.model);
    case 'anthropic':
      return anthropic(config.model);
    case 'github':
      return github(config.model);
    default:
      return openai('gpt-4o-mini');
  }
}
