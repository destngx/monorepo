/** A key button under an agent's output: what it shows, what it presses, and its spoken name. */
export interface KeyButton {
  label: string;
  keys: string[];
  name: string;
}

/**
 * Enough keys to answer an agent's menu ("1. Yes  2. Yes, always  3. No"),
 * confirm, move, switch mode, or stop it with Escape. The engine allows only
 * these keys and refuses the rest, such as ctrl+c.
 */
export const KEY_BUTTONS: readonly KeyButton[] = [
  { label: '1', keys: ['1'], name: 'Press 1' },
  { label: '2', keys: ['2'], name: 'Press 2' },
  { label: '3', keys: ['3'], name: 'Press 3' },
  { label: 'Enter', keys: ['enter'], name: 'Press Enter' },
  { label: 'Esc', keys: ['esc'], name: 'Press Escape' },
  { label: '↑', keys: ['up'], name: 'Press Up' },
  { label: '↓', keys: ['down'], name: 'Press Down' },
  { label: 'Tab', keys: ['tab'], name: 'Press Tab' },
  { label: '⇧Tab', keys: ['shift+tab'], name: 'Press Shift Tab' },
];
