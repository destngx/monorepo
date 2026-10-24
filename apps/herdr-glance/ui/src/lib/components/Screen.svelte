<script lang="ts">
  import type { Line, Span } from '$lib/ansi';
  import { paragraphs, type Block } from '$lib/screen';

  // An agent's screen as blocks: your messages as bubbles, its replies as
  // reflowed text, tool calls and the rest (status line included) as
  // terminal lines in the agent's own colours. The panel stays dark in both
  // themes, because terminal agents pick their colours for a dark terminal.
  let { blocks, agentLabel }: { blocks: Block[]; agentLabel: string } = $props();

  const FG = '#e6e1d8';
  const BG = '#1a1917';

  function css(span: Span, withBackground: boolean): string {
    const s = span.style;
    let fg = s.fg;
    let bg = withBackground ? s.bg : undefined;
    if (s.inverse) [fg, bg] = [bg ?? BG, fg ?? FG];
    const parts: string[] = [];
    if (fg) parts.push(`color:${fg}`);
    if (bg) parts.push(`background-color:${bg}`);
    if (s.bold) parts.push('font-weight:600');
    if (s.dim) parts.push('opacity:0.6');
    if (s.italic) parts.push('font-style:italic');
    if (s.underline) parts.push('text-decoration:underline');
    return parts.join(';');
  }

  const plain = (line: Line) => line.map((s) => s.text).join('');
</script>

{#snippet spans(line: Line, withBackground: boolean)}{#each line as span, i (i)}<span style={css(span, withBackground)}
      >{span.text}</span
    >{/each}{/snippet}

<div class="min-h-full px-3 py-3" style="background-color: {BG}; color: {FG}">
  {#each blocks as block, b (b)}
    {#if block.kind === 'user'}
      <section
        class="my-4 ml-8 border-2 border-accent-300 bg-accent-800 px-3 py-2 text-ink-50"
        aria-label="Your message"
      >
        <p class="mb-1 text-[0.625rem] font-semibold tracking-[0.16em] text-accent-200 uppercase">You</p>
        {#each paragraphs(block.lines) as p, i (i)}
          <p class="text-[0.9375rem] leading-snug break-words whitespace-pre-wrap">{plain(p.line) || ' '}</p>
        {/each}
      </section>
    {:else if block.kind === 'reply'}
      <section class="my-4 border-l-2 border-accent-400 pl-3" aria-label="{agentLabel} reply">
        <p class="mb-1 text-[0.625rem] font-semibold tracking-[0.16em] text-accent-300 uppercase">{agentLabel}</p>
        {#each paragraphs(block.lines) as p, i (i)}
          {#if p.line.length === 0}
            <div class="h-2"></div>
          {:else}
            <p
              class="text-[0.9375rem] leading-relaxed break-words"
              style="padding-left: {p.indent + p.hang}ch; text-indent: -{p.hang}ch"
            >
              {@render spans(p.line, false)}
            </p>
          {/if}
        {/each}
      </section>
    {:else}
      <div class="font-mono text-[12px] leading-[1.45] {block.kind === 'tool' ? 'my-2 opacity-80' : 'my-2'}">
        {#each block.lines as line, i (i)}
          <div class="min-h-[1.45em] break-words whitespace-pre-wrap">{@render spans(line, true)}</div>
        {/each}
      </div>
    {/if}
  {/each}
</div>
