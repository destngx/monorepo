// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile } from 'tailwindcss';
import { describe, expect, it } from 'vitest';

const here = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);

/** Compiles an app stylesheet that imports Tailwind and this theme, then builds the given classes. */
async function build(appCss: string, classes: string[]) {
  const compiler = await compile(appCss, {
    base: here,
    async loadStylesheet(id, base) {
      const file = id === 'tailwindcss' ? require.resolve('tailwindcss/index.css') : path.resolve(base, id);
      return { path: file, base: path.dirname(file), content: await readFile(file, 'utf8') };
    },
  });
  return { css: compiler.build(classes), sources: compiler.sources };
}

const APP = `@import 'tailwindcss';\n@import './theme.css';\n`;

describe('theme.css', () => {
  it('Given an app that imports the theme When it builds Then every shared utility exists', async () => {
    const utilities = [
      'panel',
      'panel-strong',
      'btn',
      'btn-primary',
      'btn-ghost',
      'btn-danger',
      'field',
      'label',
      'heading',
      'eyebrow',
      'tag',
      'muted',
      'link',
      'interactive',
      'row-link',
      'skeleton',
      'tabular',
      'icon-btn',
      'hit-area',
    ];
    const { css } = await build(APP, utilities);
    for (const name of utilities) expect(css, name).toContain(`.${name} {`);
  });

  it('Given the role scales When classes use them Then each colour resolves to a theme variable', async () => {
    const { css } = await build(APP, [
      'bg-accent-500',
      'text-good-700',
      'bg-warn-400',
      'text-danger-600',
      'bg-info-500',
    ]);
    for (const token of ['accent-500', 'good-700', 'warn-400', 'danger-600', 'info-500']) {
      expect(css).toContain(`--color-${token}:`);
    }
  });

  it("Given an app accent When the app's @theme follows the import Then the app's value wins", async () => {
    const { css } = await build(`${APP}@theme { --color-accent-500: #123456; }\n`, ['bg-accent-500']);
    expect(css).toContain('--color-accent-500: #123456;');
  });

  it('Given the dark class When the theme flips Then the surfaces swap and the shadow follows', async () => {
    const { css } = await build(APP, ['bg-surface', 'shadow-brut']);
    expect(css).toMatch(/\.dark\s*\{[^}]*--surface:/);
    // Tailwind inlines shadow values into the utility, so the shade variable shows there.
    expect(css).toMatch(/\.shadow-brut \{[^}]*4px 4px 0 0[^}]*var\(--shade\)/);
  });

  it('Given the components in this library When Tailwind scans sources Then the library folder is one of them', async () => {
    const { sources } = await build(APP, []);
    expect(sources.map((source) => path.resolve(source.base, source.pattern))).toContain(here);
  });
});
