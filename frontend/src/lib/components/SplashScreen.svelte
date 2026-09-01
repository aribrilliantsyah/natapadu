<script lang="ts">
  // Splash mengisi seluruh jendela kecil (460×280) sebelum jendela dibesarkan.
  // Tahapannya nyata — teksnya mengikuti pekerjaan yang benar-benar sedang berjalan,
  // bukan animasi hiasan yang jalan sendiri.
  let { stage = '', progress = 0, version = '' }: {
    stage?: string;
    progress?: number;   // 0..100
    version?: string;
  } = $props();
</script>

<div class="splash">
  <div class="body">
    <svg viewBox="0 0 1024 1024" width="72" height="72" aria-hidden="true">
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

    <div class="name">Natapadu</div>
    <div class="tag">Navigasi Master Data dan Alat Terpadu</div>
  </div>

  <!-- Bar di bagian bawah jendela, seperti splash aplikasi desktop pada umumnya -->
  <div class="foot">
    <div class="line">
      <span class="stage">{stage || 'Memuat...'}</span>
      <span class="pct mono">{Math.round(progress)}%</span>
    </div>
    <div class="track">
      <div class="fill" style="width:{Math.max(2, Math.min(100, progress))}%"></div>
    </div>
    {#if version}
      <div class="ver mono">v{version}</div>
    {/if}
  </div>
</div>

<style>
  .splash {
    position: fixed; inset: 0;
    background:
      radial-gradient(130% 100% at 50% 0%, #232a42 0%, #12141a 68%);
    display: flex; flex-direction: column;
    overflow: hidden;
    user-select: none;
  }

  .body {
    flex: 1;
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
    gap: 2px;
    padding: 0 20px;
  }
  .body svg {
    filter: drop-shadow(0 8px 22px rgba(79,110,247,0.35));
    animation: rise 420ms cubic-bezier(0.2, 0.7, 0.3, 1) both;
  }
  .name {
    margin-top: 12px;
    font-size: 22px; font-weight: 700; letter-spacing: -0.5px;
    color: #f0f1f4;
    animation: rise 420ms 60ms cubic-bezier(0.2, 0.7, 0.3, 1) both;
  }
  .tag {
    font-size: 11.5px; color: #8b90a0; text-align: center;
    animation: rise 420ms 110ms cubic-bezier(0.2, 0.7, 0.3, 1) both;
  }

  .foot {
    padding: 12px 18px 14px;
    border-top: 1px solid rgba(255,255,255,0.07);
    background: rgba(0,0,0,0.22);
  }
  .line {
    display: flex; align-items: baseline; justify-content: space-between;
    margin-bottom: 7px;
  }
  .stage {
    font-size: 11px; color: #8b90a0;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .pct {
    font-size: 10.5px; color: #4f5468;
    font-variant-numeric: tabular-nums; flex-shrink: 0; margin-left: 10px;
  }

  .track {
    height: 3px; border-radius: 2px;
    background: rgba(255,255,255,0.08);
    overflow: hidden;
  }
  .fill {
    height: 100%; border-radius: 2px;
    background: linear-gradient(90deg, #4f6ef7, #8ea2ff);
    transition: width 220ms ease;
  }

  .ver {
    margin-top: 8px;
    font-size: 10px; color: #3d4152; text-align: right;
  }

  @keyframes rise {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: none; }
  }

  @media (prefers-reduced-motion: reduce) {
    .body svg, .name, .tag { animation: none; }
    .fill { transition: none; }
  }
</style>
