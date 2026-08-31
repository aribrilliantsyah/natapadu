<script lang="ts">
  import { showToast } from '../stores/appState';
  import { X, Search, CopyCheck } from 'lucide-svelte';
  import { GetDuplicateGroups } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let { tpl, onApply, onClose }: {
    tpl: models.Template;
    // fields = kunci pembanding; values = nilai grup yang dipilih (kosong = tampilkan semua duplikat)
    onApply: (fields: string[], values: string[] | null) => void;
    onClose: () => void;
  } = $props();

  let fields = $state<string[]>(tpl.columns?.length ? [tpl.columns[0].fieldName] : []);
  let search = $state('');
  let groups = $state<models.DuplicateGroup[]>([]);
  let loading = $state(false);
  let ran = $state(false);

  const affected = $derived(groups.reduce((n, g) => n + g.count, 0));
  const labelOf = (f: string) => tpl.columns?.find(c => c.fieldName === f)?.displayName ?? f;

  function toggleField(f: string) {
    fields = fields.includes(f) ? fields.filter(x => x !== f) : [...fields, f];
  }

  async function run() {
    if (!fields.length) { showToast('Pilih minimal satu kolom pembanding', 'warning'); return; }
    loading = true;
    try {
      groups = await GetDuplicateGroups(tpl.id, fields, search, 500);
      ran = true;
    } catch (e: any) { showToast('Gagal mencari duplikat: ' + e, 'error'); groups = []; }
    finally { loading = false; }
  }
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget) onClose(); }}>
  <div class="modal" style="width:640px; height:640px; max-height:90vh;">
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">Cari Data Ganda</div>
        <div class="modal-hd-sub">Baris yang nilainya berulang pada kolom pembanding yang dipilih</div>
      </div>
      <button class="btn btn-ghost btn-icon btn-xs" onclick={onClose}><X size={14} /></button>
    </div>

    <div style="padding:14px 18px 12px; display:flex; flex-direction:column; gap:10px; flex-shrink:0;">
      <div class="field">
        <label class="field-label">
          Kolom Pembanding
          <span style="color:var(--t3); font-weight:400; text-transform:none; margin-left:5px;">
            pilih lebih dari satu untuk kunci gabungan
          </span>
        </label>
        <div style="display:flex; flex-wrap:wrap; gap:5px;">
          {#each tpl.columns ?? [] as col}
            <button type="button" class="tag {fields.includes(col.fieldName) ? 'on' : ''}"
              style="padding:4px 9px; font-size:11.5px;"
              onclick={() => toggleField(col.fieldName)}>
              {col.displayName}
            </button>
          {/each}
        </div>
      </div>

      <div style="display:flex; gap:6px;">
        <div class="search-wrap" style="flex:1;">
          <span class="search-icon"><Search size={13} /></span>
          <input class="input" type="text" bind:value={search}
            onkeydown={e => e.key === 'Enter' && run()}
            placeholder="Batasi ke nilai tertentu... (opsional)" style="padding-left:32px;" />
        </div>
        <button class="btn btn-primary" onclick={run} disabled={loading}>
          {#if loading}<span class="spin"></span>{/if}
          Cari
        </button>
      </div>
    </div>

    <div style="flex:1; overflow-y:auto; border-top:1px solid var(--line); min-height:0;">
      {#if loading}
        <div class="empty"><div class="empty-sub">Mengelompokkan data...</div></div>
      {:else if !ran}
        <div class="empty">
          <CopyCheck size={22} strokeWidth={1.5} style="color:var(--t3);" />
          <div class="empty-sub">Pilih kolom pembanding lalu tekan Cari.</div>
        </div>
      {:else if !groups.length}
        <div class="empty">
          <div class="empty-title">Tidak Ada Duplikat</div>
          <div class="empty-sub">
            Semua nilai pada {fields.map(labelOf).join(' + ')} unik{search ? ` untuk pencarian "${search}"` : ''}.
          </div>
        </div>
      {:else}
        {#each groups as g}
          <button type="button" class="dup-row" onclick={() => onApply(fields, g.values)}>
            <span class="dup-val">
              {g.values.map((v: string) => v === '' ? '(kosong)' : v).join('  ·  ')}
            </span>
            <span class="badge badge-amber mono">{g.count}×</span>
          </button>
        {/each}
      {/if}
    </div>

    <div class="modal-ft" style="justify-content:space-between;">
      <span style="font-size:11.5px; color:var(--t3);">
        {#if ran && groups.length}
          {groups.length.toLocaleString('id-ID')} grup · {affected.toLocaleString('id-ID')} baris terlibat
        {:else}
          Kunci: {fields.length ? fields.map(labelOf).join(' + ') : '—'}
        {/if}
      </span>
      <div style="display:flex; gap:6px;">
        <button class="btn btn-ghost" onclick={onClose}>Tutup</button>
        <button class="btn btn-outline" disabled={!groups.length} onclick={() => onApply(fields, null)}>
          Tampilkan Semua Baris Ganda
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .dup-row {
    width: 100%;
    display: flex; align-items: center; gap: 10px;
    padding: 9px 18px;
    border: none; border-bottom: 1px solid var(--line);
    background: transparent; color: var(--t2);
    font-family: inherit; font-size: 12.5px; text-align: left;
    cursor: pointer;
  }
  .dup-row:hover { background: var(--bg-4); color: var(--t1); }
  .dup-val { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
