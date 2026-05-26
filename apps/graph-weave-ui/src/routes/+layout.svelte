<script lang="ts">
  import './layout.css';
  import SystemHealth from '$lib/components/graphweave/SystemHealth.svelte';

  let { children } = $props();
</script>

<svelte:head>
  <title>GraphWeave Console</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
  <link
    href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=Outfit:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;700&display=swap"
    rel="stylesheet"
  />
</svelte:head>

<div
  class="min-h-screen cyber-grid text-slate-100 antialiased overflow-x-hidden selection:bg-indigo-500/30 selection:text-indigo-200 flex flex-col relative z-10"
>
  <!-- Dynamic Ambient Glowing Mesh Orbs -->
  <div
    class="fixed top-[-10%] left-[-10%] w-[50%] h-[50%] rounded-full bg-indigo-900/10 blur-[130px] pointer-events-none z-0 animate-ambient-mesh"
  ></div>
  <div
    class="fixed bottom-[-10%] right-[-10%] w-[50%] h-[50%] rounded-full bg-violet-950/10 blur-[130px] pointer-events-none z-0 animate-ambient-mesh"
    style="animation-delay: -5s;"
  ></div>
  <div
    class="fixed top-[30%] right-[10%] w-[30%] h-[30%] rounded-full bg-emerald-950/5 blur-[120px] pointer-events-none z-0 animate-ambient-mesh"
    style="animation-delay: -10s;"
  ></div>

  <!-- Sleek Top Header Bar -->
  <header
    class="w-full border-b border-slate-900/80 bg-slate-950/80 backdrop-blur-xl sticky top-0 z-40 flex items-center justify-between px-6 py-4 select-none relative"
  >
    <a href="/" class="flex items-center gap-3 group">
      <div
        class="w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 via-indigo-600 to-violet-600 flex items-center justify-center shadow-lg shadow-indigo-500/20 border border-indigo-400/20 group-hover:scale-105 transition-transform duration-300"
      >
        <svg
          width="20"
          height="20"
          class="w-5 h-5 text-white"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94-3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"
          />
        </svg>
      </div>
      <div>
        <h1
          class="text-base font-extrabold font-display tracking-tight bg-gradient-to-r from-white via-slate-100 to-slate-300 bg-clip-text text-transparent"
        >
          GraphWeave
        </h1>
        <span class="text-[9px] text-indigo-400/90 font-extrabold tracking-widest uppercase block mt-0.5"
          >Orchestration Terminal</span
        >
      </div>
    </a>

    <!-- Top Navigation Right controls -->
    <div class="flex items-center gap-4">
      <span class="hidden sm:inline-flex text-[9px] text-slate-500 font-medium tracking-wide">
        Core: <span class="text-indigo-400 font-bold font-mono">v0.1.0-alpha</span>
      </span>
      <SystemHealth />
    </div>
  </header>

  <!-- Main Content Panel -->
  <main class="flex-1 flex flex-col min-w-0 w-full relative z-10">
    {@render children()}
  </main>
</div>

<style>
  :global(body) {
    background-color: #030408;
    color: #f1f5f9;
  }
  :global(.font-display) {
    font-family: 'Outfit', sans-serif;
  }
  :global(.font-sans) {
    font-family: 'Inter', sans-serif;
  }
  :global(.font-mono) {
    font-family: 'JetBrains Mono', monospace;
  }
</style>
