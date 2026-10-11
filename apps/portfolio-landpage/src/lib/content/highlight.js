import { createHighlighter } from 'shiki';

const THEMES = { light: 'github-light', dark: 'vitesse-dark' };
const LANGS = ['ts', 'js', 'go', 'python', 'swift', 'svelte', 'bash', 'sh', 'json', 'yaml', 'sql', 'diff', 'text'];

/** @type {Promise<import('shiki').Highlighter> | undefined} */
let highlighterPromise;

/**
 * Escapes characters that Svelte would otherwise treat as template syntax.
 * @param {string} html
 */
export const escapeSvelte = (html) =>
  html.replace(/[{}`\\]/g, (char) => ({ '{': '&#123;', '}': '&#125;', '`': '&#96;', '\\': '&#92;' })[char] ?? char);

/**
 * mdsvex highlighter: renders fenced code with Shiki dual themes (switched via CSS variables).
 * @param {string} code
 * @param {string | null | undefined} lang
 */
export async function highlightCode(code, lang) {
  highlighterPromise ??= createHighlighter({ themes: Object.values(THEMES), langs: LANGS });
  const highlighter = await highlighterPromise;
  const language = lang && highlighter.getLoadedLanguages().includes(lang) ? lang : 'text';
  const html = highlighter.codeToHtml(code, { lang: language, themes: THEMES, defaultColor: false });
  return `{@html \`${escapeSvelte(html)}\`}`;
}
