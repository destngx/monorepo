import { loadSystemPromptAsset } from '../loader';

/**
 * Loads the Lộc Phát Tài identity prompt from Google Sheets.
 */
export async function getIdentityPrompt(): Promise<string> {
  return loadSystemPromptAsset('identity');
}
