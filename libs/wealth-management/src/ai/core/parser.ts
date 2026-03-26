/**
 * Securely extracts and parses JSON from AI-generated text.
 * Handles markdown backticks, conversational preamble, and common formatting artifacts.
 * Includes a robust heuristic for fixing trailing commas and incomplete/truncated JSON structures.
 */
export function extractAndParseJSON<T>(text: string): T {
  let jsonStr = '';
  try {
    // 1. Locate the outer structure (Object or Array)
    const startIndexObj = text.indexOf('{');
    const endIndexObj = text.lastIndexOf('}');
    const startIndexArr = text.indexOf('[');
    const endIndexArr = text.lastIndexOf(']');

    const isObject = startIndexObj !== -1 && (startIndexArr === -1 || startIndexObj < startIndexArr);

    if (isObject) {
      // For an object, we take from first '{' to last '}'
      // But if the stream is truncated, there might not be a last '}'
      const effectiveEnd = endIndexObj !== -1 && endIndexObj > startIndexObj ? endIndexObj + 1 : text.length;
      jsonStr = text.substring(startIndexObj, effectiveEnd);
    } else if (startIndexArr !== -1) {
      const effectiveEnd = endIndexArr !== -1 && endIndexArr > startIndexArr ? endIndexArr + 1 : text.length;
      jsonStr = text.substring(startIndexArr, effectiveEnd);
    } else {
      throw new Error('No JSON structure found in AI response');
    }

    // 2. Initial cleanup
    jsonStr = jsonStr.trim();

    // 3. Heuristic: Fix common LLM-generated JSON errors
    // a. Remove trailing commas in objects and arrays
    jsonStr = jsonStr.replace(/,\s*([}\]])/g, '$1');

    try {
      return JSON.parse(jsonStr) as T;
    } catch (parseError: any) {
      // 4. Secondary Attempt: Try to repair truncated JSON
      try {
        const repaired = repairTruncatedJSON(jsonStr);
        return JSON.parse(repaired) as T;
      } catch {
        // Fall back to original error message
        console.error('[AI Parser Error]:', parseError.message);
        const snippet = text.length > 500 ? text.substring(0, 500) + '...' : text;
        console.debug('[AI Raw Text]:', snippet);
        throw parseError;
      }
    }
  } catch (error: any) {
    if (error instanceof SyntaxError) {
      console.error('[AI Parser Error]: JSON Syntax Error:', error.message);
    } else {
      console.error('[AI Parser Error]:', error.message);
    }
    throw new Error(`AI response was not valid JSON: ${error.message}`);
  }
}

/**
 * Attempts to repair truncated JSON strings by closing unclosed brackets and quotes.
 */
function repairTruncatedJSON(json: string): string {
  let inString = false;
  let escaped = false;
  const stack: string[] = [];

  for (let i = 0; i < json.length; i++) {
    const char = json[i];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (char === '\\') {
      escaped = true;
      continue;
    }
    if (char === '"') {
      inString = !inString;
      continue;
    }
    if (inString) continue;

    if (char === '{' || char === '[') {
      stack.push(char === '{' ? '}' : ']');
    } else if (char === '}' || char === ']') {
      if (stack.length > 0 && stack[stack.length - 1] === char) {
        stack.pop();
      }
    }
  }

  let result = json;

  // If we ended mid-string, close the string
  if (inString) {
    // If it ends with backslash, remove it
    if (result.endsWith('\\')) result = result.slice(0, -1);
    result += '"';
  }

  // Remove trailing comma if exists after string closure
  result = result.trim().replace(/,$/, '');

  // Close all brackets in reverse order
  while (stack.length > 0) {
    const closing = stack.pop();
    result += closing;
  }

  return result;
}
