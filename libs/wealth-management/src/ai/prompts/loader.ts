import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export function loadPrompt(domain: string, name: string): string {
  try {
    const filePath = path.join(__dirname, 'assets', domain, `${name}.md`);
    return fs.readFileSync(filePath, 'utf-8').trim();
  } catch (error) {
    console.error(`Failed to load prompt: ${domain}/${name}`, error);
    return '';
  }
}

/**
 * Loads a prompt from the tasks directory
 */
export function loadTaskPrompt(name: string): string {
  return loadPrompt('tasks', name);
}

/**
 * Loads a prompt from the system directory
 */
export function loadSystemPromptAsset(name: string): string {
  return loadPrompt('system', name);
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
