<script lang="ts">
  // The page background from the portfolio landing page (apps/portfolio-landpage
  // Aurora): slow drifting colour blobs under a halftone dot screen. The colours
  // are the theme's own (--aurora-* and --dot in theme.css), kept faint so cards
  // stay calm. Mount it once, in the root layout.
  // Transform-only animation; it stands still for reduced motion.
</script>

<div class="pointer-events-none fixed inset-0 -z-10 overflow-hidden" aria-hidden="true" data-aurora>
  <div class="blob blob-1"></div>
  <div class="blob blob-2"></div>
  <div class="blob blob-3"></div>
  <div class="blob blob-4"></div>
  <div class="halftone absolute inset-0"></div>
</div>

<style>
  .blob {
    position: absolute;
    border-radius: 9999px;
    filter: blur(90px);
    will-change: transform;
  }

  .blob-1 {
    top: -20vh;
    left: -10vw;
    width: 60vmax;
    height: 60vmax;
    background: radial-gradient(circle, var(--aurora-1), transparent 65%);
    animation: drift-1 26s ease-in-out infinite;
  }

  .blob-2 {
    top: 10vh;
    right: -20vw;
    width: 55vmax;
    height: 55vmax;
    background: radial-gradient(circle, var(--aurora-2), transparent 65%);
    animation: drift-2 22s ease-in-out infinite;
  }

  .blob-3 {
    bottom: -30vh;
    left: 10vw;
    width: 50vmax;
    height: 50vmax;
    background: radial-gradient(circle, var(--aurora-3), transparent 65%);
    animation: drift-3 30s ease-in-out infinite;
  }

  .blob-4 {
    top: 40vh;
    left: 40vw;
    width: 30vmax;
    height: 30vmax;
    background: radial-gradient(circle, var(--aurora-4), transparent 65%);
    animation: drift-2 18s ease-in-out infinite reverse;
  }

  .halftone {
    background-image: radial-gradient(var(--dot) 1.2px, transparent 1.4px);
    background-size: 9px 9px;
  }

  @keyframes drift-1 {
    50% {
      transform: translate(30vw, 40vh) scale(1.15);
    }
  }

  @keyframes drift-2 {
    50% {
      transform: translate(-35vw, 20vh) scale(0.9);
    }
  }

  @keyframes drift-3 {
    50% {
      transform: translate(25vw, -35vh) scale(1.1);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .blob {
      animation: none;
    }
  }
</style>
