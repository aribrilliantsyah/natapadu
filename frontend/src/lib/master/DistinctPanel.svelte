<script lang="ts">
  import { showToast } from '../stores/appState';
  import { X, Search, Layers } from 'lucide-svelte';
  import { GetDistinctValues } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let { tpl, field: initialField = '', onPick, onClose }: {
    tpl: models.Template;
    field?: string;
    // values = seluruh daftar nilai kolom ini, dipakai untuk menelusuri satu per satu
    onPick: (field: string, value: string, values: string[]) => void;
    onClose: () => void;
  } = $props();

  let field = $state(initialField || tpl.columns?.[0]?.fieldName || '');
  let search = $state('');
  let items = $state<models.DistinctValue[]>([]);
  let loading = $state(false);

  const totalShown = $derived(items.reduce((n, it) => n + it.count, 0));

  async function load() {
    if (!field) return;
    loading = true;
    try {
      items = await GetDistinctValues(tpl.id, field, search, 1000);
    } catch (e: any) { showToast('Gagal ambil nilai unik: ' + e, 'error'); items = []; }
    finally { loading = false; }
  }

  function pick(v: string) {
    onPick(field, v, items.map(i => i.value));
  }

  $effect(() => { field; load(); });
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget) onClose(); }}>
  <div class="modal" style="width:520px; height:600px; max-height:88vh;">
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">Nilai Unik</div>
        <div class="modal-hd-sub">Pilih satu nilai untuk memfilter tabel — lalu telusuri sisanya satu per satu</div>
      </div>
      <button class="btn btn-ghost btn-icon btn-xs" onclick={onClose}><X size={14} /></button>
    </div>

    <div style="padding:14px 18px 10px; display:flex; flex-direction:column; gap:10px; flex-shrink:0;">
      <div class="field">
        <label class="field-label">Kolom</label>
        <select class="select" bind:value={field}>
          {#each tpl.columns ?? [] as col}
            <option value={col.fieldName}>{col.displayName}</option>
          {/each}
        </select>
      </div>

      <div class="search-wrap">
        <span class="search-icon"><Search size={13} /></span>
        <input class="input" type="text" bind:value={search}
          onkeydown={e => e.key === 'Enter' && load()}
          placeholder="Saring nilai... (Enter)" style="padding-left:32px;" />
      </div>
    </div>

    <div style="flex:1; overflow-y:auto; border-top:1px solid var(--line); min-height:0;">
      {#if loading}
        <div class="empty"><div class="empty-sub">Menghitung nilai unik...</div></div>
      {:else if !items.length}
        <div class="empty">
          <Layers size={22} strokeWidth={1.5} style="color:var(--t3);" />
          <div class="empty-sub">
            {search ? `Tidak ada nilai yang cocok dengan "${search}".` : 'Kolom ini belum punya data.'}
          </div>
        </div>
      {:else}
        {#each items as it}
          <button type="button" class="dv-row" onclick={() => pick(it.value)}>
            <span class="dv-val">{it.value === '' ? '(kosong)' : it.value}</span>
            <span class="badge badge-gray mono">{it.count.toLocaleString()}</span>
          </button>
        {/each}
      {/if}
    </div>

    <div class="modal-ft" style="justify-content:space-between;">
      <span style="font-size:11.5px; color:var(--t3);">
        {items.length.toLocaleString()} nilai unik · {totalShown.toLocaleString()} baris tercakup
      </span>
      <button class="btn btn-ghost" onclick={onClose}>Tutup</button>
    </div>
  </div>
</div>

<style>
  .dv-row {
    width: 100%;
    display: flex; align-items: center; gap: 10px;
    padding: 9px 18px;
    border: none;
    border-bottom: 1px solid var(--line);
    background: transparent;
    color: var(--t2);
    font-family: inherit;
    font-size: 12.5px;
    text-align: left;
    cursor: pointer;
  }
  .dv-row:hover { background: var(--bg-4); color: var(--t1); }
  .dv-val {
    flex: 1; min-width: 0;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
</style>
