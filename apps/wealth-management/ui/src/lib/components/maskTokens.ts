import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

// One clickable mask token for every hidden value (WM-UI-2, WM-UI-15): clicking
// it toggles the global privacy mask, like a Money amount. The click never
// reaches an enclosing link, row or card.
//
// Inside a link (or any other interactive element) a nested button is invalid
// HTML and nested interactive content is an ARIA violation, so there the token
// is a plain span that handles the pointer click only and stays out of the tab
// order: the link stays the single focusable element, and keyboard users reveal
// with the header eye or any standalone token, which are real buttons.

export const MASK_TOKEN_LABEL = 'Hidden amount, click to reveal';
export const MASK_TOKEN_TITLE = 'Show amounts';

/** Hover, pressed and focus-visible feedback shared by every token. */
export const MASK_TOKEN_CLASS =
  'tabular -mx-0.5 cursor-pointer rounded px-0.5 whitespace-nowrap transition-colors hover:bg-slate-900/10 ' +
  'active:bg-slate-900/20 focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:outline-none ' +
  'dark:hover:bg-white/15 dark:active:bg-white/25';

/** Marks a token in rendered HTML, for the delegated click handler. */
export const MASK_TOKEN_ATTR = 'data-mask-toggle';

/** Toggles the mask from a token's click without triggering what encloses it. */
export function toggleMask(event: Event): void {
  event.preventDefault();
  event.stopPropagation();
  mask.toggle();
}

/**
 * Attachment that toggles on click with a native listener. Svelte delegates
 * `onclick` to the root, where stopPropagation is too late for native
 * listeners on ancestors; a listener on the token itself stops it in time.
 */
export function maskToggle(node: HTMLElement): () => void {
  node.addEventListener('click', toggleMask);
  return () => node.removeEventListener('click', toggleMask);
}

const buttonToken =
  `<button type="button" ${MASK_TOKEN_ATTR} class="${MASK_TOKEN_CLASS}" ` +
  `aria-label="${MASK_TOKEN_LABEL}" title="${MASK_TOKEN_TITLE}">${MASK}</button>`;
const linkToken = `<span ${MASK_TOKEN_ATTR} class="${MASK_TOKEN_CLASS}" title="${MASK_TOKEN_TITLE}">${MASK}</span>`;

// Escaped text never holds a raw < or >, so every <...> run is a tag.
const tag = /(<[^>]*>)/;

/**
 * Wraps every MASK in the text of already rendered and escaped HTML in a token
 * element. Only the exact MASK string in text (never inside a tag, such as a
 * link's href) is replaced, by constant markup, so no input reaches the markup.
 * Inside links the token is the span form (see above).
 */
export function tokenizeMaskedHtml(html: string): string {
  if (!html.includes(MASK)) return html;
  let isInLink = false;
  return html
    .split(tag)
    .map((part, index) => {
      if (index % 2) {
        if (/^<a\b/i.test(part)) isInLink = true;
        else if (/^<\/a>/i.test(part)) isInLink = false;
        return part;
      }
      return part.replaceAll(MASK, isInLink ? linkToken : buttonToken);
    })
    .join('');
}

/** Delegated click handler for a container of tokenized HTML. */
export function onMaskTokenClick(event: MouseEvent): void {
  const target = event.target;
  if (!(target instanceof Element)) return;
  const token = target.closest(`[${MASK_TOKEN_ATTR}]`);
  if (token && event.currentTarget instanceof Element && event.currentTarget.contains(token)) toggleMask(event);
}

/** Attachment for a container of tokenized HTML: one native listener for all its tokens. */
export function maskTokenClicks(node: HTMLElement): () => void {
  node.addEventListener('click', onMaskTokenClick);
  return () => node.removeEventListener('click', onMaskTokenClick);
}
