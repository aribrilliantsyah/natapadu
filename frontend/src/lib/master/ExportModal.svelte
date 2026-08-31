<script lang="ts">
  import { activeQuery, showToast } from '../stores/appState';
  import { Download, Folder, CheckCircle2, X, FileSpreadsheet, FileText, FileType } from 'lucide-svelte';
  import { SelectDirectory, ExportData } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let { tpl, onClose }: { tpl: models.Template; onClose: () => void } = $props();

  const grid = $derived($activeQuery);
  const gridMatch = $derived(grid.templateId === tpl.id);
  const hasFilter = $derived(gridMatch && (grid.filters.length > 0 || !!grid.searchTerm));
  const hasSelection = $derived(gridMatch && grid.selectedRowIds.length > 0);

  // Default ke cakupan tersempit yang tersedia — biasanya itu yang dimaksud user
  let scope = $state<'ALL'|'FILTERED'|'SELECTED'>(
    $activeQuery.templateId === tpl.id && $activeQuery.selectedRowIds.length > 0 ? 'SELECTED'
    : $activeQuery.templateId === tpl.id && ($activeQuery.filters.length > 0 || !!$activeQuery.searchTerm) ? 'FILTERED'
    : 'ALL'
  );
  const FORMATS = [
    { v: 'XLSX', label: 'Excel',        ext: '.xlsx', icon: FileSpreadsheet, hint: 'Format Excel modern (.xlsx)' },
    { v: 'CSV',  label: 'CSV',          ext: '.csv',  icon: FileText,        hint: 'Teks polos, UTF-8 dengan BOM' },
    { v: 'ODS',  label: 'OpenDocument', ext: '.ods',  icon: FileType,        hint: 'LibreOffice / OpenOffice (.ods)' },
  ] as const;

  let format = $state<'XLSX'|'CSV'|'ODS'>('XLSX');
  const baseName = `Export_${tpl.name.replace(/\s+/g,'_')}_${new Date().toISOString().slice(0,10)}`;
  let fname = $state(baseName + '.xlsx');

  // Ekstensi mengikuti format yang dipilih (backend juga mengoreksi, ini supaya terlihat di UI)
  function pickFormat(f: 'XLSX'|'CSV'|'ODS') {
    format = f;
    const ext = FORMATS.find(x => x.v === f)!.ext;
    fname = fname.replace(/\.(xlsx|csv|ods)$/i, '') + ext;
  }
  let saveDir = $state('');
  let selCols = $state<string[]>(tpl.columns?.map(c => c.fieldName) ?? []);
  let exporting = $state(false), done = $state<{path:string;count:number}|null>(null);

  async function pickDir() {
    const d = await SelectDirectory('Pilih Folder Output'); if (d) saveDir = d;
  }

  function togCol(fn: string) {
    selCols = selCols.includes(fn) ? selCols.filter(c => c !== fn) : [...selCols, fn];
  }

  async function doExport() {
    if (!selCols.length) { showToast('Pilih minimal 1 kolom', 'warning'); return; }
    exporting = true; done = null;
    try {
      const q = $activeQuery;
      const fromGrid = q.templateId === tpl.id;
      const res = await ExportData({
        templateId: tpl.id,
        format,
        scope,
        columns: selCols,
        outputFilename: fname,
        searchTerm: scope === 'FILTERED' && fromGrid ? q.searchTerm : '',
        filters:    scope === 'FILTERED' && fromGrid ? q.filters : [],
        filterLogic: fromGrid ? q.filterLogic : 'AND',
        sortBy:     fromGrid ? q.sortBy : '',
        sortOrder:  fromGrid ? q.sortOrder : 'ASC',
        selectedRowIds: scope === 'SELECTED' && fromGrid ? q.selectedRowIds : [],
      } as any, saveDir);
      done = { path: res.filePath, count: res.rowCount };
      showToast(`${res.rowCount.toLocaleString()} baris diekspor ke ${res.format}`, 'success');
    } catch (e: any) { showToast('Export gagal: ' + e, 'error'); }
    finally { exporting = false; }
  }
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget) onClose(); }}>
  <div class="modal" style="width:640px; max-height:88vh;">
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">Export — {tpl.name}</div>
        <div class="modal-hd-sub">Simpan hasil olahan kembali ke file Excel</div>
      </div>
      <button class="btn btn-ghost btn-icon btn-xs" onclick={onClose}><X size={14} /></button>
    </div>

    <div class="modal-body" style="display:flex; flex-direction:column; gap:12px;">
      <div class="field">
        <label class="field-label">Format Berkas</label>
        <div style="display:grid; grid-template-columns:repeat(3,1fr); gap:6px;">
          {#each FORMATS as f}
            {@const Icon = f.icon}
            <button type="button" class="fmt-btn {format === f.v ? 'on' : ''}" title={f.hint} onclick={() => pickFormat(f.v)}>
              <Icon size={15} />
              <span style="font-size:12.5px; font-weight:600;">{f.label}</span>
              <span class="mono" style="font-size:10.5px; opacity:0.7;">{f.ext}</span>
            </button>
          {/each}
        </div>
      </div>

      <div style="display:grid; grid-template-columns:1fr 1fr; gap:12px;">
        <div class="field">
          <label class="field-label">Cakupan Export</label>
          <select class="select" bind:value={scope}>
            <option value="ALL">Semua Data ({tpl.recordCount?.toLocaleString() ?? 0} baris)</option>
            <option value="FILTERED" disabled={!hasFilter}>
              Hasil Filter{hasFilter ? ` (${grid.totalRows.toLocaleString()} baris)` : ' — belum ada filter aktif'}
            </option>
            <option value="SELECTED" disabled={!hasSelection}>
              Baris Terpilih{hasSelection ? ` (${grid.selectedRowIds.length.toLocaleString()} baris)` : ' — belum ada baris dicentang'}
            </option>
          </select>
        </div>
        <div class="field">
          <label class="field-label">Nama File Output</label>
          <input class="input" type="text" bind:value={fname} placeholder="Export.xlsx" />
        </div>
      </div>

      <div class="field">
        <label class="field-label">Folder Simpan</label>
        <div style="display:flex; gap:6px;">
          <input class="input" type="text" readonly value={saveDir || 'Default (~/Downloads)'}
            style="flex:1; color:var(--t2);" />
          <button class="btn btn-outline" style="flex-shrink:0;" onclick={pickDir}>
            <Folder size={13} />
          </button>
        </div>
      </div>

      {#if scope === 'ALL'}
        <div class="inset" style="padding:9px 11px; font-size:11.5px; color:var(--t3); line-height:1.5;">
          Ingin sebagian saja? Tutup dialog ini, pasang filter atau centang baris di tabel, lalu buka Export lagi.
        </div>
      {:else}
        <div class="inset" style="padding:9px 11px; font-size:11.5px; color:var(--t2); line-height:1.5;">
          {#if scope === 'FILTERED'}
            Mewarisi dari tabel: {grid.filters.length} kondisi filter{grid.searchTerm ? ` + pencarian "${grid.searchTerm}"` : ''}
            → {grid.totalRows.toLocaleString()} baris.
          {:else}
            {grid.selectedRowIds.length.toLocaleString()} baris tercentang akan diekspor.
          {/if}
        </div>
      {/if}

      <hr class="sep" />

      <div>
        <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:8px;">
          <div style="font-size:12.5px; font-weight:600; color:var(--t1);">
            Pilih Kolom
            <span style="font-size:11.5px; color:var(--t3); font-weight:400; margin-left:6px;">
              {selCols.length}/{tpl.columns?.length ?? 0} dipilih
            </span>
          </div>
          <div style="display:flex; gap:6px;">
            <button class="btn btn-ghost btn-xs" onclick={() => selCols = tpl.columns?.map(c => c.fieldName) ?? []}>Semua</button>
            <button class="btn btn-ghost btn-xs" onclick={() => selCols = []}>Hapus</button>
          </div>
        </div>
        <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(190px,1fr)); gap:5px;">
          {#each tpl.columns ?? [] as col}
            {@const on = selCols.includes(col.fieldName)}
            <label style="
              display:flex; align-items:center; gap:7px; padding:6px 9px;
              border-radius:7px; cursor:pointer;
              background:{on?'var(--accent-dim)':'var(--bg-4)'};
              border:1px solid {on?'rgba(79,110,247,0.3)':'transparent'};
              font-size:12.5px; color:{on?'var(--accent-text)':'var(--t2)'};
              transition:all 80ms;
            ">
              <input class="checkbox" type="checkbox" checked={on} onchange={() => togCol(col.fieldName)} />
              <span style="overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">{col.displayName}</span>
            </label>
          {/each}
        </div>
      </div>

      {#if done}
        <div class="inset" style="padding:11px; border-color:rgba(52,211,153,0.25); display:flex; gap:8px; align-items:flex-start;">
          <CheckCircle2 size={15} style="color:var(--green); flex-shrink:0; margin-top:1px;" />
          <div style="min-width:0;">
            <div style="font-size:12.5px; font-weight:600; color:var(--green);">
              {done.count.toLocaleString()} baris tersimpan
            </div>
            <div class="mono" style="font-size:11px; color:var(--t3); word-break:break-all;">{done.path}</div>
          </div>
        </div>
      {/if}
    </div>

    <div class="modal-ft">
      <button class="btn btn-ghost" onclick={onClose}>Tutup</button>
      <button class="btn btn-primary" onclick={doExport} disabled={exporting}>
        {#if exporting}<span class="spin"></span>{/if}
        <Download size={13} /> {exporting ? 'Menulis file...' : `Export ke ${format}`}
      </button>
    </div>
  </div>
</div>

<style>
  .fmt-btn {
    display: flex; flex-direction: column; align-items: center; gap: 3px;
    padding: 9px 6px;
    border-radius: 8px;
    border: 1px solid var(--line);
    background: var(--bg-4);
    color: var(--t2);
    cursor: pointer;
    font-family: inherit;
    transition: all 100ms ease;
  }
  .fmt-btn:hover:not(.on) { color: var(--t1); border-color: var(--line-2); }
  .fmt-btn.on {
    background: var(--accent-dim);
    border-color: rgba(79,110,247,0.45);
    color: var(--accent-text);
  }
</style>
