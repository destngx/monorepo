<script lang="ts">
  import { hobbies, milestones, photographyUrl, profile, roles, skillGroups, socials } from '$lib/data/profile';
  import { reveal } from '$lib/motion/reveal';
  import ChapterHeading from '$lib/ui/ChapterHeading.svelte';
  import Hero from '$lib/ui/Hero.svelte';
  import PostRow from '$lib/ui/PostRow.svelte';
  import ProjectCard from '$lib/ui/ProjectCard.svelte';
  import Seo from '$lib/ui/Seo.svelte';

  let { data } = $props();

  let hasCopied = $state(false);

  const copyEmail = async () => {
    try {
      await navigator.clipboard.writeText(profile.email);
      hasCopied = true;
      setTimeout(() => (hasCopied = false), 1800);
    } catch (error) {
      console.warn('Clipboard unavailable, falling back to mailto', error);
      window.location.href = `mailto:${profile.email}`;
    }
  };

  // Alternate wide panels so the grid reads like a manga page: wide, narrow / narrow, wide.
  const isWide = (index: number) => index % 4 === 0 || index % 4 === 3;
</script>

<Seo />

<Hero />

<!-- Chapter 1: Work -->
<section id="work" class="py-16" aria-labelledby="work-title">
  <ChapterHeading number="01" kicker="Selected work" title="Things I've built" id="work-title" />

  <div class="grid gap-6 md:grid-cols-3">
    {#each data.featured as project, index (project.slug)}
      <div class={isWide(index) ? 'md:col-span-2' : ''} use:reveal={{ delay: (index % 2) * 90 }}>
        <ProjectCard {project} isLarge={isWide(index)} />
      </div>
    {/each}
  </div>

  <div class="mt-10 flex justify-end">
    <a class="btn btn-ghost" href="/projects">All {data.projectCount} projects →</a>
  </div>
</section>

<!-- Chapter 2: Story -->
<section id="story" class="py-16" aria-labelledby="story-title">
  <ChapterHeading number="02" kicker="The story so far" title="From back end to the pipeline" id="story-title" />

  <div class="grid gap-10 lg:grid-cols-[1fr_1.4fr]">
    <div use:reveal>
      <p class="text-lg leading-relaxed">{profile.intro}</p>

      <ol class="relative mt-10 space-y-6 border-l-[3px] border-rule pl-6">
        {#each milestones as milestone, index (index)}
          <li class="relative">
            <span
              class="absolute top-1.5 -left-[2.05rem] h-4 w-4 rotate-45 border-[3px] border-rule bg-accent"
              aria-hidden="true"
            ></span>
            <span class="font-mono text-sm font-bold">{milestone.year}</span>
            <p class="text-ink-soft">{milestone.text}</p>
          </li>
        {/each}
      </ol>
    </div>

    <div class="space-y-6">
      {#each roles as role, index (role.title)}
        <article class="panel p-6" use:reveal={{ delay: index * 80 }}>
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 class="font-display text-2xl">{role.title}</h3>
              {#if role.company}<p class="font-mono text-sm text-ink-soft">{role.company}</p>{/if}
            </div>
            <span class="chip -rotate-3 bg-ink text-paper">{role.period}</span>
          </div>
          <p class="mt-3 font-medium">{role.summary}</p>
          <ul class="mt-3 space-y-1.5 text-ink-soft">
            {#each role.highlights as highlight (highlight)}
              <li class="flex gap-2"><span aria-hidden="true">▸</span>{highlight}</li>
            {/each}
          </ul>
          <ul class="mt-4 flex flex-wrap gap-1.5" aria-label="Tech used">
            {#each role.stack as tech (tech)}
              <li class="chip">{tech}</li>
            {/each}
          </ul>
        </article>
      {/each}
    </div>
  </div>
</section>

<!-- Chapter 3: Toolbox -->
<section id="toolbox" class="py-16" aria-labelledby="toolbox-title">
  <ChapterHeading number="03" kicker="Toolbox" title="What's in the bag" id="toolbox-title" />

  <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
    {#each skillGroups as group, index (group.title)}
      <div class="panel panel-hover p-6" use:reveal={{ delay: (index % 3) * 70 }}>
        <h3 class="font-display text-lg">{group.title}</h3>
        <ul class="mt-4 flex flex-wrap gap-2">
          {#each group.skills as skill (skill)}
            <li class="chip text-sm">{skill}</li>
          {/each}
        </ul>
      </div>
    {/each}
  </div>
</section>

<!-- Chapter 4: Off the clock -->
<section id="off-the-clock" class="py-16" aria-labelledby="hobbies-title">
  <ChapterHeading number="04" kicker="Off the clock" title="When the laptop's closed" id="hobbies-title" />

  <ul class="grid grid-cols-2 gap-5 md:grid-cols-3">
    {#each hobbies as hobby, index (hobby.name)}
      <li
        class="panel panel-hover group halftone relative overflow-hidden p-5 md:p-6"
        style:rotate="{index % 2 === 0 ? -0.6 : 0.8}deg"
        use:reveal={{ delay: (index % 3) * 70 }}
      >
        <span
          class="sfx absolute -right-1 -bottom-2 text-5xl opacity-90 transition-transform duration-300 group-hover:scale-125 md:text-6xl"
          aria-hidden="true">{hobby.sfx}</span
        >
        <h3 class="relative font-display text-lg md:text-xl">{hobby.name}</h3>
        <p class="relative mt-2 pr-8 text-sm text-ink-soft md:text-base">{hobby.note}</p>
      </li>
    {/each}
  </ul>

  <p class="mt-8 text-ink-soft">
    I also take photos now and then; they live in a little <a
      class="ink-link"
      href={photographyUrl}
      target="_blank"
      rel="noopener">photo gallery</a
    >
    I built on Cloudinary.
  </p>
</section>

<!-- Chapter 5: Writing -->
<section id="writing" class="py-16" aria-labelledby="writing-title">
  <ChapterHeading number="05" kicker="Notes" title="Latest writing" id="writing-title" />

  {#if data.posts.length}
    <div use:reveal>
      {#each data.posts as post (post.slug)}
        <PostRow {post} />
      {/each}
    </div>
    <div class="mt-8 flex justify-end">
      <a class="btn btn-ghost" href="/blog">All writing →</a>
    </div>
  {:else}
    <p class="bubble inline-block text-lg" use:reveal>First issue is still at the printer. Check back soon.</p>
  {/if}
</section>

<!-- Final page: Contact -->
<section id="contact" class="py-16" aria-labelledby="contact-title">
  <div class="panel halftone relative overflow-hidden p-8 pt-24 md:p-14" use:reveal>
    <div class="speedlines absolute inset-0 opacity-10" aria-hidden="true"></div>
    <span class="sfx absolute top-4 right-6 rotate-12 text-5xl md:text-8xl" aria-hidden="true">バーン</span>

    <p class="chapter-label relative">Final chapter</p>
    <h2 id="contact-title" class="relative mt-2 max-w-2xl font-display text-4xl leading-tight md:text-6xl">
      Got a gnarly pipeline? Let's talk.
    </h2>
    <p class="relative mt-6 max-w-xl text-lg text-ink-soft">
      Always happy to hear about new DevOps opportunities, cloud infrastructure projects, or just nerd out about a side
      project.
    </p>

    <div class="relative mt-10 flex flex-wrap items-center gap-4">
      <a class="btn btn-primary" href="mailto:{profile.email}">Email me</a>
      <button type="button" class="btn btn-ghost" onclick={copyEmail} data-cursor={hasCopied ? 'got it' : 'copy'}>
        {hasCopied ? 'Copied!' : 'Copy address'}
      </button>
      <span class="sr-only" aria-live="polite">{hasCopied ? 'Email address copied to clipboard' : ''}</span>
    </div>

    <ul class="relative mt-10 flex flex-wrap gap-6 font-mono text-sm font-semibold">
      {#each socials as social (social.label)}
        <li>
          <a
            class="ink-link"
            href={social.href}
            rel="me noopener"
            target={social.href.startsWith('http') ? '_blank' : undefined}
          >
            {social.label} <span class="text-ink-soft">{social.handle}</span>
          </a>
        </li>
      {/each}
    </ul>
  </div>
</section>
