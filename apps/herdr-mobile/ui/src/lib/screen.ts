/** A rule longer than this many characters is shortened to it. */
const RULE_WIDTH = 16;
const RULE = /^([─━═\-_]){20,}$/u;

/**
 * Makes an agent's terminal output readable at phone width. The engine sends
 * unwrapped lines, which the page wraps; this trims the padding a terminal
 * leaves and shortens full-width rules, which would wrap into several lines.
 */
export function tidyScreen(text: string): string {
  const lines = text.split('\n').map((line) => {
    const trimmed = line.trimEnd();
    const rule = RULE.exec(trimmed.trim());
    return rule ? rule[1].repeat(RULE_WIDTH) : trimmed;
  });
  const out: string[] = [];
  for (const line of lines) {
    if (line === '' && (out.length === 0 || out[out.length - 1] === '')) continue;
    out.push(line);
  }
  while (out.length > 0 && out[out.length - 1] === '') out.pop();
  return out.join('\n');
}
