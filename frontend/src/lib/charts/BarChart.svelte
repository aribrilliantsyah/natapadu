<script lang="ts">
  import type { models } from '../../../wailsjs/go/models';

  // Batang horizontal untuk membandingkan besaran antar kategori.
  // Satu hue (besaran sudah dikodekan oleh panjang batang), label langsung di tiap baris.
  let { data, unit = 'baris', color = 'var(--chart-1)', empty = 'Belum ada data.' }: {
    data: models.ChartPoint[];
    unit?: string;
    color?: string;
    empty?: string;
  } = $props();

  const max = $derived(Math.max(1, ...data.map(d => d.value)));
  const fmt = (n: number) => n.toLocaleString('id-ID');
</script>

{#if !data?.length}
  <div class="empty" style="padding:32px;"><div class="empty-sub">{empty}</div></div>
{:else}
  <div class="rows">
    {#each data as d}
      <div class="row" title="{d.label}: {fmt(d.value)} {unit}">
        <div class="label">{d.label}</div>
        <div class="track">
          <div class="fill" style="width:{Math.max(2, (d.value / max) * 100)}%; background:{color};"></div>
        </div>
        <div class="value mono">{fmt(d.value)}</div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .rows { display: flex; flex-direction: column; gap: 9px; }
  .row {
    display: grid;
    grid-template-columns: minmax(70px, 130px) 1fr auto;
    align-items: center;
    gap: 10px;
  }
  .label {
    font-size: 12px; color: var(--t2);
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .track {
    height: 9px; border-radius: 5px;
    background: rgba(255,255,255,0.05);
    overflow: hidden;
  }
  /* Ujung data membulat 4px, menempel pada garis dasar kiri */
  .fill {
    height: 100%;
    border-radius: 0 4px 4px 0;
    transition: width 260ms ease;
  }
  .value {
    font-size: 11.5px; color: var(--t2);
    font-variant-numeric: tabular-nums;
    min-width: 56px; text-align: right;
  }
</style>
