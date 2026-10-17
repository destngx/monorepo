import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import { MASK_TOKEN_LABEL, onMaskTokenClick, tokenizeMaskedHtml } from './maskTokens';
import Masked from './Masked.svelte';
import MaskedText from './MaskedText.svelte';
import Money from './Money.svelte';

afterEach(() => {
  mask.hide();
  document.body.innerHTML = '';
});

const TEXT = 'Spent 1.500.000 ₫ on food and 300k on coffee, 12% more';

/** A host element in the document, e.g. a link or a clickable row, that records clicks. */
function host(tag: 'a' | 'div') {
  const element = document.createElement(tag);
  if (tag === 'a') element.setAttribute('href', '/elsewhere');
  const clicked = vi.fn();
  element.addEventListener('click', clicked);
  document.body.append(element);
  return { element, clicked };
}

describe('MaskedText', () => {
  it('Given masked text When it renders Then each amount is a labelled token button and the rest is text', () => {
    const { container } = render(MaskedText, { text: TEXT });

    const tokens = screen.getAllByRole('button', { name: MASK_TOKEN_LABEL });
    expect(tokens).toHaveLength(2);
    expect(tokens.every((token) => token.textContent === MASK && token.title === 'Show amounts')).toBe(true);
    expect(container.textContent).toBe(`Spent ${MASK} on food and ${MASK} on coffee, 12% more`);
  });

  it('Given masked text When a token is clicked Then every amount reveals and the text equals the original', async () => {
    const { container } = render(MaskedText, { text: TEXT });
    render(Money, { value: 42_000 });

    await fireEvent.click(screen.getAllByRole('button', { name: MASK_TOKEN_LABEL })[1]);

    expect(mask.revealed).toBe(true);
    expect(container.textContent).toBe(TEXT);
    expect(screen.getByText('42.000 ₫')).toBeTruthy();
  });

  it('Given a token in a clickable row When it is clicked Then the row does not see the click', async () => {
    const row = host('div');
    render(MaskedText, { target: row.element, props: { text: TEXT } });

    await fireEvent.click(screen.getAllByRole('button')[0]);

    expect(row.clicked).not.toHaveBeenCalled();
    expect(mask.revealed).toBe(true);
  });

  it('Given a token When it has focus Then it is a native button, so Enter and Space activate it', () => {
    render(MaskedText, { text: TEXT });
    const token = screen.getAllByRole('button')[0] as HTMLButtonElement;

    token.focus();

    expect(document.activeElement).toBe(token);
    expect(token.tagName).toBe('BUTTON');
    expect(token.type).toBe('button');
    expect(token.disabled).toBe(false);
  });

  it('Given text inside a link When its token is clicked Then the mask toggles without navigating', async () => {
    const link = host('a');
    render(MaskedText, { target: link.element, props: { text: TEXT, inLink: true } });

    // No button nested in the link: the token is a pointer-only span and the link stays the one focusable element.
    expect(link.element.querySelector('button')).toBeNull();
    const token = link.element.querySelector('[data-mask-toggle]') as HTMLElement;
    expect(token.tagName).toBe('SPAN');
    expect(token.tabIndex).toBe(-1);

    const isDefaultAllowed = await fireEvent.click(token);

    expect(isDefaultAllowed).toBe(false);
    expect(link.clicked).not.toHaveBeenCalled();
    expect(mask.revealed).toBe(true);
    expect(link.element.textContent).toBe(TEXT);
  });

  it('Given text without amounts When masked Then it renders unchanged with no token', () => {
    const { container } = render(MaskedText, { text: 'Two bills due, 40% of budget' });
    expect(container.textContent).toBe('Two bills due, 40% of budget');
    expect(screen.queryByRole('button')).toBeNull();
  });
});

describe('Masked', () => {
  it('Given a masked name When its token is clicked Then names and amounts reveal globally', async () => {
    render(Masked, { text: 'Lender A' });
    render(Money, { value: 1_000 });
    expect(screen.queryByText('Lender A')).toBeNull();

    await fireEvent.click(screen.getAllByRole('button', { name: MASK_TOKEN_LABEL })[0]);

    expect(screen.getByText('Lender A')).toBeTruthy();
    expect(screen.getByText('1.000 ₫')).toBeTruthy();
  });

  it('Given a masked name in a clickable row When its token is clicked Then the row does not see the click', async () => {
    const row = host('div');
    render(Masked, { target: row.element, props: { text: 'Lender A' } });

    await fireEvent.click(screen.getByRole('button'));

    expect(row.clicked).not.toHaveBeenCalled();
    expect(row.element.textContent).toBe('Lender A');
  });
});

describe('tokenizeMaskedHtml', () => {
  it('Given rendered HTML When it holds the mask Then each one becomes a token button', () => {
    const html = tokenizeMaskedHtml(`<p>Spent <strong>${MASK}</strong></p>`);
    const root = document.createElement('div');
    root.innerHTML = html;
    const token = root.querySelector('strong > button[data-mask-toggle]') as HTMLButtonElement;
    expect(token.type).toBe('button');
    expect(token.getAttribute('aria-label')).toBe(MASK_TOKEN_LABEL);
    expect(token.textContent).toBe(MASK);
  });

  it('Given a link When its label or href holds the mask Then the label gets a span token and the tag is untouched', () => {
    const link = `<a href="https://a.vn/${MASK}" target="_blank" rel="noopener noreferrer">`;
    const html = tokenizeMaskedHtml(`${link}cost ${MASK}</a> then ${MASK}`);
    const root = document.createElement('div');
    root.innerHTML = html;
    const anchor = root.querySelector('a') as HTMLAnchorElement;
    expect(anchor.getAttribute('href')).toBe(`https://a.vn/${MASK}`);
    expect(anchor.querySelector('button')).toBeNull();
    expect(anchor.querySelector('span[data-mask-toggle]')?.textContent).toBe(MASK);
    expect(root.querySelectorAll(':scope > button[data-mask-toggle]')).toHaveLength(1);
  });

  it('Given HTML without the mask When tokenized Then it is returned as is', () => {
    expect(tokenizeMaskedHtml('<p>a &lt;b&gt;</p>')).toBe('<p>a &lt;b&gt;</p>');
  });

  it('Given a delegated container When a non-token is clicked Then the mask stays', () => {
    const root = document.createElement('div');
    root.innerHTML = '<p>plain</p>';
    root.addEventListener('click', onMaskTokenClick);
    document.body.append(root);

    (root.querySelector('p') as HTMLElement).click();

    expect(mask.revealed).toBe(false);
  });
});
