import { getKnowledgeContent } from '../../services/sheets/content';

/**
 * Loads a knowledge document from the Google Sheets "Knowledge" tab.
 * Relies on the cached content map (populated by prefetchAllContent at startup).
 */
export async function loadKnowledge(name: string): Promise<string> {
  const content = await getKnowledgeContent(name);
  if (!content) {
    console.error(`Knowledge not found in sheet: ${name}`);
    return '';
  }
  return content.trim();
}
