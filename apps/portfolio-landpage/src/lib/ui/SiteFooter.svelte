<script lang="ts">
  import { profile, socials } from '$lib/data/profile';

  const year = new Date().getFullYear();
</script>

<footer class="mx-auto mt-24 max-w-6xl px-4 pb-10 sm:px-6">
  <div
    class="flex flex-col gap-4 border-t-[3px] border-dashed border-rule pt-6 sm:flex-row sm:items-center sm:justify-between"
  >
    <p class="font-mono text-xs text-ink-soft">
      © {year}
      {profile.name}. Inked with SvelteKit. <a class="ink-link" href="/rss.xml">RSS</a>
    </p>
    <ul class="flex gap-4 font-mono text-xs font-semibold uppercase tracking-wider">
      {#each socials as social (social.label)}
        <li>
          <a
            class="ink-link"
            href={social.href}
            rel="me noopener"
            target={social.href.startsWith('http') ? '_blank' : undefined}>{social.label}</a
          >
        </li>
      {/each}
    </ul>
  </div>
  <p class="mt-6 text-center font-display text-sm text-ink-soft" aria-hidden="true">完 / fin</p>
</footer>
