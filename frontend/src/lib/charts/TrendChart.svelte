<script lang="ts">
  import type { models } from '../../../wailsjs/go/models';

  // Kolom bertumpuk per hari: sukses + gagal.
  // Pasangan warna ini berada di ambang bawah pemisahan CVD, jadi identitas TIDAK
  // pernah hanya lewat warna — ada legenda, jarak 2px antar segmen, dan angka di tooltip.
  let { data, colorOk = 'var(--chart-ok)', colorFail = 'var(--chart-fail)' }: {
    data: models.ChartPoint[];
    colorOk?: string;
    colorFail?: string;
  } = $props();

  const max = $derived(Math.max(1, ...data.map(d => d.value + d.secondary)));
  const totalOk = $derived(data.reduce((n, d) => n + d.value, 0));
  const totalFail = $derived(data.reduce((n, d) => n + d.secondary, 0));

  const fmt = (n: number) => n.toLocaleString('id-ID');
  const dayLabel = (iso: string) => {
    const d = new Date(iso + 'T00:00:00');
    return isNaN(d.getTime()) ? iso : d.toLocaleDateString('id-ID', { day: '2-digit', month: 'short' });
  };
</script>

<div class="wrap">
  <div class="legend">
    <span class="lg"><i style="background:{colorOk};"></i>Sukses <b class="mono">{fmt(totalOk)}</b></span>
    <span class="lg"><i style="background:{colorFail};"></i>Gagal <b class="mono">{fmt(totalFail)}</b></span>
  </div>

  {#if totalOk + totalFail === 0}
    <div class="empty" style="padding:26px;">
      <div class="empty-sub">Belum ada import dalam 14 hari terakhir.</div>
    </div>
  {:else}
    <div class="plot">
      {#each data as d}
        {@const total = d.value + d.secondary}
        <div class="col" title="{dayLabel(d.label)} — sukses {fmt(d.value)}, gagal {fmt(d.secondary)}">
          <div class="stack">
            {#if d.secondary > 0}
              <div class="seg" style="height:{(d.secondary / max) * 100}%; background:{colorFail};"></div>
            {/if}
            {#if d.value > 0}
              <div class="seg base" style="height:{(d.value / max) * 100}%; background:{colorOk};"></div>
            {/if}
            {#if total === 0}
              <div class="seg zero"></div>
            {/if}
          </div>
          <div class="tick">{dayLabel(d.label).split(' ')[0]}</div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .wrap { display: flex; flex-direction: column; gap: 10px; }

  .legend { display: flex; gap: 14px; align-items: center; }
  .lg {
    display: inline-flex; align-items: center; gap: 6px;
    font-size: 11.5px; color: var(--t2);
  }
  .lg i { width: 9px; height: 9px; border-radius: 3px; flex-shrink: 0; }
  .lg b { color: var(--t1); font-weight: 600; }

  .plot {
    display: flex; align-items: flex-end; gap: 4px;
    height: 132px;
  }
  .col {
    flex: 1; min-width: 0;
    display: flex; flex-direction: column;
    align-items: center; gap: 5px;
    height: 100%;
  }
  .stack {
    width: 100%;
    flex: 1;
    display: flex; flex-direction: column;
    justify-content: flex-end;
    gap: 2px;                      /* jarak permukaan antar segmen bertumpuk */
  }
  .seg { width: 100%; border-radius: 3px 3px 0 0; min-height: 3px; }
  .seg.base { border-radius: 0; }
  .seg.zero {
    height: 3px; border-radius: 2px;
    background: rgba(255,255,255,0.07);
  }
  .col:hover .seg { filter: brightness(1.15); }

  .tick {
    font-size: 10px; color: var(--t3);
    font-variant-numeric: tabular-nums;
  }
</style>
