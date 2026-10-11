<!--
  Drifting aurora blobs from the v1 site, rebuilt with transform-only animation
  and a halftone screen on top so it reads like a printed colour layer.
-->
<div class="pointer-events-none fixed inset-0 -z-10 overflow-hidden" aria-hidden="true">
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
</style>
