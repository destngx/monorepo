/**
 * Securely extracts and parses JSON from AI-generated text.
 * Handles markdown backticks, conversational preamble, and common formatting artifacts.
 */
export function extractAndParseJSON<T>(text: string): T {
  try {
    // Look for the first '{' and the last '}'
    const startIndex = text.indexOf('{');
    const endIndex = text.lastIndexOf('}');

    if (startIndex === -1 || endIndex === -1 || endIndex < startIndex) {
      // Try to look for an array if object not found
      const arrayStart = text.indexOf('[');
      const arrayEnd = text.lastIndexOf(']');
      
      if (arrayStart !== -1 && arrayEnd !== -1 && arrayEnd > arrayStart) {
        return JSON.parse(text.substring(arrayStart, arrayEnd + 1)) as T;
      }
      
      throw new Error("No JSON structure found in AI response");
    }

    const jsonStr = text.substring(startIndex, endIndex + 1).trim();
    return JSON.parse(jsonStr) as T;
  } catch (error: any) {
    console.error("[AI Parser Error]:", error.message);
    console.debug("[AI Raw Text]:", text);
    throw new Error(`AI response was not valid JSON: ${error.message}`);
  }
}
