import { getPromptContent } from '../../services/sheets/content';

/**
 * Loads a prompt from the Google Sheets "Prompts" tab.
 * Relies on the cached content map (populated by prefetchAllContent at startup).
 */
export async function loadPrompt(domain: string, name: string): Promise<string> {
  const content = await getPromptContent(domain, name);
  if (!content) {
    console.error(`Prompt not found in sheet: ${domain}/${name}`);
    return '';
  }
  return content.trim();
}

/**
 * Loads a prompt from the tasks domain
 */
export async function loadTaskPrompt(name: string): Promise<string> {
  return loadPrompt('tasks', name);
}

/**
 * Loads a prompt from the system domain
 */
export async function loadSystemPromptAsset(name: string): Promise<string> {
  return loadPrompt('system', name);
}

/**
 * Loads a prompt from the actions domain (used for short task-specific prompts in API routes)
 */
export async function loadActionPrompt(name: string): Promise<string> {
  return loadPrompt('actions', name);
}

/**
 * Utility to replace {{key}} placeholders in a string
 */
export function replacePlaceholders(template: string, data: Record<string, any>): string {
  let result = template;
  for (const [key, value] of Object.entries(data)) {
    const stringValue = typeof value === 'string' ? value : JSON.stringify(value);
    result = result.replace(new RegExp(`{{${key}}}`, 'g'), stringValue);
  }
  return result;
}
