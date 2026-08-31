<script lang="ts">
  let { done = false }: { done?: boolean } = $props();
</script>

<!-- Splash desktop: tetap terlihat sampai sesi selesai diperiksa, lalu memudar -->
<div class="splash" class:out={done}>
  <div class="mark">
    <svg viewBox="0 0 1024 1024" width="96" height="96" aria-hidden="true">
      <defs>
        <linearGradient id="sp-tool" x1="0.1" y1="0" x2="0.9" y2="1">
          <stop offset="0%" stop-color="#a9b8ff"/>
          <stop offset="100%" stop-color="#4f6ef7"/>
        </linearGradient>
        <mask id="sp-wrench">
          <rect width="1024" height="1024" fill="#000"/>
          <circle cx="512" cy="312" r="156" fill="#fff"/>
          <rect x="450" y="300" width="124" height="470" rx="62" fill="#fff"/>
          <circle cx="512" cy="312" r="80" fill="#000"/>
          <polygon points="512,312 372,96 652,96" fill="#000"/>
        </mask>
      </defs>
      <g fill="#4f6ef7" fill-opacity="0.32">
        <rect x="228" y="640" width="82" height="150" rx="28"/>
        <rect x="342" y="566" width="82" height="224" rx="28"/>
        <rect x="456" y="492" width="82" height="298" rx="28"/>
      </g>
      <g transform="rotate(38 512 470) translate(0 60)">
        <rect width="1024" height="1024" fill="url(#sp-tool)" mask="url(#sp-wrench)"/>
      </g>
    </svg>
  </div>

  <div class="name">Natapadu</div>
  <div class="tag">Navigasi Master Data dan Alat Terpadu</div>

  <div class="bar"><div class="bar-fill"></div></div>
</div>

<style>
  .splash {
    position: fixed; inset: 0; z-index: 9999;
    background: radial-gradient(120% 90% at 50% 0%, #232a42 0%, var(--bg, #0d0e11) 62%);
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
    gap: 6px;
    transition: opacity 320ms ease, visibility 320ms ease;
  }
  .splash.out { opacity: 0; visibility: hidden; }

  .mark {
    animation: rise 520ms cubic-bezier(0.2, 0.7, 0.3, 1) both;
    filter: drop-shadow(0 10px 28px rgba(79,110,247,0.35));
  }
  .name {
    margin-top: 14px;
    font-size: 25px; font-weight: 700; letter-spacing: -0.6px;
    color: var(--t1, #f0f1f4);
    animation: rise 520ms 60ms cubic-bezier(0.2, 0.7, 0.3, 1) both;
  }
  .tag {
    font-size: 12.5px; color: var(--t2, #8b90a0);
    animation: rise 520ms 120ms cubic-bezier(0.2, 0.7, 0.3, 1) both;
  }

  .bar {
    margin-top: 26px;
    width: 168px; height: 3px; border-radius: 2px;
    background: rgba(255,255,255,0.08);
    overflow: hidden;
  }
  .bar-fill {
    height: 100%; width: 42%; border-radius: 2px;
    background: linear-gradient(90deg, transparent, #4f6ef7, transparent);
    animation: sweep 1.1s ease-in-out infinite;
  }

  @keyframes rise {
    from { opacity: 0; transform: translateY(10px); }
    to   { opacity: 1; transform: none; }
  }
  @keyframes sweep {
    0%   { transform: translateX(-140%); }
    100% { transform: translateX(340%); }
  }

  @media (prefers-reduced-motion: reduce) {
    .mark, .name, .tag { animation: none; }
    .bar-fill { animation: none; width: 100%; }
  }
</style>
