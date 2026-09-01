<script lang="ts">
  import { onMount } from 'svelte';
  import { activeQuery, showToast } from '../stores/appState';
  import {
    ArrowLeft, Search, Trash2, ChevronLeft, ChevronRight, RefreshCw, Plus, X,
    SlidersHorizontal, Download, Upload, Settings2, Pencil, Columns3, Layers, CopyCheck
  } from 'lucide-svelte';
  import {
    GetTemplateByID, QueryData, DeleteRow, BulkDeleteRows, TruncateDataset, GetDataRow
  } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';
  import TemplateDesigner from './TemplateDesigner.svelte';
  import ImportModal from './ImportModal.svelte';
  import ExportModal from './ExportModal.svelte';
  import RowEditor from './RowEditor.svelte';
  import DistinctPanel from './DistinctPanel.svelte';
  import DuplicatePanel from './DuplicatePanel.svelte';

  let { tpl, startWith = 'data', onBack, onChanged }: {
    tpl: models.Template;
    startWith?: 'data' | 'import';
    onBack: () => void;
    onChanged: (tpl: models.Template) => void;
  } = $props();

  let cur = $state<models.Template>(tpl);
  const ready = $derived((cur.columns?.length ?? 0) > 0);

  let page = $state(1), size = $state(50);
  let q = $state(''), sortBy = $state(''), sortDir = $state<'ASC'|'DESC'>('ASC');
  let filters = $state<models.FilterCondition[]>([]);
  let result = $state<models.QueryResponse | null>(null);
  let loading = $state(false), selIds = $state<number[]>([]);
  let filterOpen = $state(false);

  // Dialog aktif — semua aksi workspace tinggal di satu halaman ini
  let dialog = $state<'' | 'structure' | 'import' | 'export' | 'row' | 'distinct' | 'duplicate'>(startWith === 'import' ? 'import' : '');
  let filterLogic = $state<'AND' | 'OR'>('AND');
  let editRowId = $state(0), editRowData = $state<Record<string, any>>({});

  const OPS = [
    {v:'equals',l:'= equals'},{v:'not_equals',l:'≠ not equals'},{v:'contains',l:'contains'},
    {v:'not_contains',l:'not contains'},{v:'starts_with',l:'starts with'},{v:'ends_with',l:'ends with'},
    {v:'is_empty',l:'is empty'},{v:'is_not_empty',l:'not empty'},
    {v:'gt',l:'> lebih dari'},{v:'gte',l:'≥ minimal'},{v:'lt',l:'< kurang dari'},{v:'lte',l:'≤ maksimal'},
    {v:'between',l:'antara (2 nilai)'},
    {v:'in_list',l:'salah satu dari'},{v:'not_in_list',l:'bukan salah satu dari'},
    {v:'is_duplicate',l:'nilainya ganda'},{v:'is_not_duplicate',l:'nilainya unik'},
  ];

  // Operator yang tidak butuh input nilai sama sekali
  const NO_VALUE = ['is_empty','is_not_empty','is_duplicate','is_not_duplicate'];
  const LIST_OPS = ['in_list','not_in_list'];

  async function refreshTpl() {
    try {
      const fresh = await GetTemplateByID(cur.id);
      if (fresh) { cur = fresh; onChanged(fresh); }
    } catch {}
  }

  async function run() {
    if (!ready) { loading = false; return; }
    loading = true;
    try {
      result = await QueryData({ templateId: cur.id, page, pageSize: size, searchTerm: q, sortBy, sortOrder: sortDir, filters, filterLogic } as any);
      selIds = [];
      publishQuery();
    } catch (e: any) { showToast('Query error: ' + e, 'error'); }
    finally { loading = false; }
  }

  // Terbitkan filter aktif supaya Export bisa memakai cakupan FILTERED / SELECTED
  function publishQuery() {
    activeQuery.set({
      templateId: cur.id, searchTerm: q, filters, filterLogic, sortBy, sortOrder: sortDir,
      selectedRowIds: selIds, totalRows: result?.totalRows ?? 0,
    });
  }

  function toggleSort(f: string) {
    if (sortBy === f) sortDir = sortDir === 'ASC' ? 'DESC' : 'ASC';
    else { sortBy = f; sortDir = 'ASC'; }
    page = 1; run();
  }

  function toggle(id: number) {
    selIds = selIds.includes(id) ? selIds.filter(x => x !== id) : [...selIds, id];
    publishQuery();
  }
  function toggleAll() {
    if (!result?.data) return;
    const all = result.data.map(r => Number(r._row_id));
    selIds = selIds.length === all.length ? [] : all;
    publishQuery();
  }

  async function del(id: number) {
    if (!confirm('Hapus baris ini?')) return;
    try { await DeleteRow(cur.id, id); showToast('Dihapus', 'success'); await reloadAll(); }
    catch (e: any) { showToast('Gagal: ' + e, 'error'); }
  }

  async function bulkDel() {
    if (!confirm(`Hapus ${selIds.length} baris?`)) return;
    try { const n = await BulkDeleteRows(cur.id, selIds); showToast(`${n} baris dihapus`, 'success'); await reloadAll(); }
    catch (e: any) { showToast('Gagal: ' + e, 'error'); }
  }

  async function truncate() {
    if (!confirm(`Kosongkan SEMUA data "${cur.name}"? Struktur kolom tetap dipertahankan.`)) return;
    try { await TruncateDataset(cur.id); showToast('Dataset dikosongkan', 'success'); page = 1; await reloadAll(); }
    catch (e: any) { showToast('Gagal: ' + e, 'error'); }
  }

  function addF() {
    if (!cur.columns?.length) return;
    filters = [...filters, { fieldName: cur.columns[0].fieldName, operator: 'contains', value: '' } as models.FilterCondition];
  }

  // Telusur per nilai unik: satu kolom, daftar nilainya, maju/mundur satu per satu
  let stepField = $state('');
  let stepValues = $state<string[]>([]);
  let stepIndex = $state(-1);

  function applyStep(i: number) {
    if (i < 0 || i >= stepValues.length) return;
    stepIndex = i;
    filters = [{ fieldName: stepField, operator: 'equals', value: stepValues[i] } as models.FilterCondition];
    q = '';
    page = 1;
    run();
  }

  function pickDistinct(field: string, value: string, values: string[]) {
    dialog = '';
    stepField = field;
    stepValues = values;
    applyStep(values.indexOf(value));
  }

  // Hasil panel duplikat diterjemahkan jadi kondisi filter biasa,
  // supaya paginasi, sortir, dan Export cakupan "Hasil Filter" ikut mengikuti.
  function applyDuplicate(fields: string[], values: string[] | null) {
    dialog = '';
    filterLogic = 'AND';
    if (values === null) {
      filters = [{
        fieldName: fields[0],
        operator: 'is_duplicate',
        value: fields.slice(1).join(','),
      } as models.FilterCondition];
    } else {
      filters = fields.map((f, i) => ({
        fieldName: f, operator: 'equals', value: values[i],
      })) as models.FilterCondition[];
    }
    q = '';
    stepField = ''; stepValues = []; stepIndex = -1;
    filterOpen = true;
    page = 1;
    run();
  }

  function clearStep() {
    stepField = ''; stepValues = []; stepIndex = -1;
    filters = []; page = 1; run();
  }

  const stepLabel = $derived(
    cur.columns?.find(c => c.fieldName === stepField)?.displayName ?? stepField
  );

  async function openRow(id: number) {
    try {
      editRowData = await GetDataRow(cur.id, id);
      editRowId = id;
      dialog = 'row';
    } catch (e: any) { showToast('Gagal memuat baris: ' + e, 'error'); }
  }

  function newRow() { editRowId = 0; editRowData = {}; dialog = 'row'; }

  async function reloadAll() { await refreshTpl(); await run(); }

  onMount(run);
</script>

<div style="display:flex; flex-direction:column; height:100%; overflow:hidden;">
  <!-- Header workspace: identitas + semua aksi -->
  <div class="topbar">
    <button class="btn btn-ghost btn-icon btn-xs" title="Kembali ke daftar workspace" onclick={onBack}>
      <ArrowLeft size={15} />
    </button>
    <span class="topbar-title" style="flex:0 0 auto;">{cur.name}</span>
    <span class="badge badge-gray">{(result?.totalRows ?? cur.recordCount ?? 0).toLocaleString()} baris</span>
    <span class="badge badge-gray">{cur.columns?.length ?? 0} kolom</span>

    <div style="flex:1;"></div>

    <button class="btn btn-ghost btn-xs" onclick={() => dialog = 'structure'}>
      <Settings2 size={12} /> Struktur
    </button>
    <button class="btn btn-outline btn-xs" disabled={!ready} onclick={() => dialog = 'import'}>
      <Upload size={12} /> Import Excel
    </button>
    <button class="btn btn-outline btn-xs" disabled={!ready} onclick={newRow}>
      <Plus size={12} /> Tambah Data
    </button>
    <button class="btn btn-outline btn-xs" disabled={!ready} onclick={() => dialog = 'export'}>
      <Download size={12} /> Export
    </button>
  </div>

  {#if !ready}
    <!-- Workspace ada tapi strukturnya belum diatur -->
    <div class="empty" style="flex:1;">
      <div style="
        width:56px; height:56px; border-radius:15px;
        background:var(--bg-3); border:1px solid var(--line-2);
        display:flex; align-items:center; justify-content:center;
        color:var(--t3); margin-bottom:8px;
      ">
        <Columns3 size={26} strokeWidth={1.5} />
      </div>
      <div class="empty-title">Struktur Belum Diatur</div>
      <div class="empty-sub">
        Tentukan dulu kolom-kolom workspace ini. Setelah tersimpan, Anda bisa mengunduh
        template Excel, mengimpor file, atau menambah baris satu per satu.
      </div>
      <button class="btn btn-primary" style="margin-top:12px;" onclick={() => dialog = 'structure'}>
        <Settings2 size={13} /> Atur Struktur Kolom
      </button>
    </div>

  {:else}
    <!-- Toolbar tabel -->
    <div class="topbar" style="height:44px; background:var(--bg-3);">
      <div class="search-wrap" style="flex:1; max-width:300px;">
        <span class="search-icon"><Search size={13} /></span>
        <input class="input input-sm" type="text" bind:value={q}
          onkeydown={e => e.key === 'Enter' && (page = 1, run())}
          placeholder="Cari data... (Enter)" style="padding-left:30px; height:28px;" />
      </div>

      <button class="btn btn-xs {filterOpen || filters.length > 0 ? 'btn-primary' : 'btn-outline'}"
        onclick={() => filterOpen = !filterOpen}>
        <SlidersHorizontal size={12} />
        Filter {filters.length > 0 ? `(${filters.length})` : ''}
      </button>

      <button class="btn btn-outline btn-xs" title="Lihat nilai unik sebuah kolom"
        onclick={() => dialog = 'distinct'}>
        <Layers size={12} /> Nilai Unik
      </button>

      <button class="btn btn-outline btn-xs" title="Cari baris dengan nilai berulang"
        onclick={() => dialog = 'duplicate'}>
        <CopyCheck size={12} /> Data Ganda
      </button>

      <button class="btn btn-ghost btn-icon btn-xs" onclick={run} title="Muat ulang">
        <RefreshCw size={13} style={loading ? 'animation:_spin 0.65s linear infinite;' : ''} />
      </button>

      {#if selIds.length > 0}
        <div class="topbar-sep"></div>
        <span style="font-size:12px; color:var(--t2);">{selIds.length} dipilih</span>
        <button class="btn btn-danger btn-xs" onclick={bulkDel}><Trash2 size={12} /> Hapus</button>
      {/if}

      <div style="flex:1;"></div>
      <button class="btn btn-ghost btn-xs" style="color:var(--red); font-size:12px;" onclick={truncate}>
        Kosongkan
      </button>
    </div>

    {#if stepIndex >= 0 && stepValues.length}
      <!-- Menelusuri nilai unik satu per satu -->
      <div class="filter-strip" style="flex-direction:row; align-items:center; gap:8px;">
        <span style="font-size:12px; color:var(--t3);">{stepLabel}:</span>
        <span class="badge badge-blue" style="max-width:280px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">
          {stepValues[stepIndex] === '' ? '(kosong)' : stepValues[stepIndex]}
        </span>
        <div style="flex:1;"></div>
        <button class="btn btn-ghost btn-icon btn-xs" title="Nilai sebelumnya"
          disabled={stepIndex <= 0} onclick={() => applyStep(stepIndex - 1)}>
          <ChevronLeft size={14} />
        </button>
        <span class="mono" style="font-size:11.5px; color:var(--t2); min-width:70px; text-align:center;">
          {stepIndex + 1} / {stepValues.length}
        </span>
        <button class="btn btn-ghost btn-icon btn-xs" title="Nilai berikutnya"
          disabled={stepIndex >= stepValues.length - 1} onclick={() => applyStep(stepIndex + 1)}>
          <ChevronRight size={14} />
        </button>
        <button class="btn btn-ghost btn-xs" onclick={() => dialog = 'distinct'}>Ganti kolom</button>
        <button class="btn btn-ghost btn-xs" style="color:var(--red);" onclick={clearStep}>Selesai</button>
      </div>
    {/if}

    {#if filterOpen}
      <div class="filter-strip">
        <div style="display:flex; align-items:center; justify-content:space-between;">
          <div style="display:flex; align-items:center; gap:8px;">
            <span style="font-size:12px; font-weight:600; color:var(--t1);">Filter</span>
            <div class="seg" style="padding:2px;">
              {#each ['AND','OR'] as mode}
                <button class="seg-btn {filterLogic === mode ? 'on' : ''}" style="height:22px; padding:0 10px; font-size:11px;"
                  title={mode === 'AND' ? 'Semua kondisi harus terpenuhi' : 'Cukup salah satu kondisi terpenuhi'}
                  onclick={() => { filterLogic = mode as 'AND'|'OR'; if (filters.length) { page = 1; run(); } }}>{mode}</button>
              {/each}
            </div>
          </div>
          <div style="display:flex; gap:6px;">
            <button class="btn btn-outline btn-xs" onclick={addF}><Plus size={11}/> Tambah</button>
            {#if filters.length > 0}
              <button class="btn btn-primary btn-xs" onclick={() => { page = 1; run(); }}>Terapkan</button>
              <button class="btn btn-ghost btn-xs" onclick={() => { filters = []; page = 1; run(); }}>Reset</button>
            {/if}
          </div>
        </div>
        {#if filters.length === 0}
          <span style="font-size:12px; color:var(--t3); font-style:italic;">Tambahkan kondisi filter di atas.</span>
        {:else}
          {#each filters as f, i}
            <div class="filter-row">
              <select class="select select-sm" bind:value={f.fieldName} style="width:150px;">
                {#each cur.columns as col}<option value={col.fieldName}>{col.displayName}</option>{/each}
              </select>
              <select class="select select-sm" bind:value={f.operator} style="width:130px;">
                {#each OPS as op}<option value={op.v}>{op.l}</option>{/each}
              </select>
              {#if f.operator === 'between'}
                <input class="input input-sm" type="text" bind:value={f.value} placeholder="Dari..." style="width:110px;" />
                <span style="font-size:11px; color:var(--t3);">s/d</span>
                <input class="input input-sm" type="text" bind:value={f.valueTo} placeholder="Sampai..." style="width:110px;" />
              {:else if LIST_OPS.includes(f.operator)}
                <input class="input input-sm" type="text" bind:value={f.value}
                  placeholder="Nilai dipisah koma..." style="width:260px;" />
              {:else if f.operator === 'is_duplicate' || f.operator === 'is_not_duplicate'}
                <select class="select select-sm" bind:value={f.value} style="width:200px;">
                  <option value="">— hanya kolom ini —</option>
                  {#each cur.columns.filter(c => c.fieldName !== f.fieldName) as col}
                    <option value={col.fieldName}>+ {col.displayName}</option>
                  {/each}
                </select>
              {:else if !NO_VALUE.includes(f.operator)}
                <input class="input input-sm" type="text" bind:value={f.value} placeholder="Nilai..." style="width:180px;" />
              {/if}
              <button class="btn btn-ghost btn-icon btn-xs" onclick={() => filters = filters.filter((_, idx) => idx !== i)}>
                <X size={12} />
              </button>
            </div>
          {/each}
        {/if}
      </div>
    {/if}

    <!-- Tabel -->
    <div style="flex:1; overflow:auto;">
      <table class="tbl" style="min-width:max-content;">
        <thead>
          <tr>
            <th style="width:36px; padding:8px 10px;">
              <input class="checkbox" type="checkbox"
                checked={selIds.length > 0 && selIds.length === result?.data?.length}
                onchange={toggleAll} />
            </th>
            <th class="mono" style="width:50px; text-align:right; color:var(--t3);">ID</th>
            {#each cur.columns as col}
              <th class="sort" onclick={() => toggleSort(col.fieldName)}>
                {col.displayName}
                {#if sortBy === col.fieldName}
                  <span style="margin-left:3px; opacity:0.5;">{sortDir === 'ASC' ? '↑' : '↓'}</span>
                {/if}
              </th>
            {/each}
            <th style="width:64px;"></th>
          </tr>
        </thead>
        <tbody>
          {#if loading}
            <tr><td colspan={100} style="padding:50px; text-align:center; color:var(--t3);">Mengeksekusi query...</td></tr>
          {:else if !result?.data?.length}
            <tr>
              <td colspan={100} style="padding:0;">
                <div class="empty" style="padding:56px 24px;">
                  <div class="empty-title">
                    {filters.length || q ? 'Tidak Ada Baris Cocok' : 'Workspace Masih Kosong'}
                  </div>
                  <div class="empty-sub">
                    {filters.length || q
                      ? 'Ubah atau reset filter untuk melihat data lain.'
                      : 'Import file Excel, atau tambah baris satu per satu.'}
                  </div>
                  {#if !(filters.length || q)}
                    <div style="display:flex; gap:6px; margin-top:12px;">
                      <button class="btn btn-primary btn-xs" onclick={() => dialog = 'import'}>
                        <Upload size={12} /> Import Excel
                      </button>
                      <button class="btn btn-outline btn-xs" onclick={newRow}>
                        <Plus size={12} /> Tambah Data
                      </button>
                    </div>
                  {/if}
                </div>
              </td>
            </tr>
          {:else}
            {#each result.data as row}
              {@const rid = Number(row._row_id)}
              <tr class={selIds.includes(rid) ? 'sel' : ''}>
                <td style="padding:6px 10px; text-align:center;">
                  <input class="checkbox" type="checkbox" checked={selIds.includes(rid)} onchange={() => toggle(rid)} />
                </td>
                <td class="mono" style="text-align:right; color:var(--t3); padding:6px 10px;">{row._row_id}</td>
                {#each cur.columns as col}
                  <td data-sel style="padding:6px 12px; max-width:220px; overflow:hidden; text-overflow:ellipsis;">
                    {row[col.fieldName] !== null && row[col.fieldName] !== undefined ? row[col.fieldName] : ''}
                  </td>
                {/each}
                <td style="padding:4px 8px; white-space:nowrap;">
                  <button class="btn btn-ghost btn-icon btn-xs" title="Ubah baris" onclick={() => openRow(rid)}>
                    <Pencil size={12} />
                  </button>
                  <button class="btn btn-ghost btn-icon btn-xs" title="Hapus baris"
                    style="color:var(--red);" onclick={() => del(rid)}>
                    <Trash2 size={12} />
                  </button>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <!-- Footer paginasi -->
    <div style="
      height:36px; padding:0 14px;
      border-top:1px solid var(--line);
      background:var(--bg-2);
      display:flex; align-items:center; gap:10px;
      font-size:12px; color:var(--t2);
      flex-shrink:0;
    ">
      <span>
        <span class="mono" style="color:var(--t1); font-weight:600;">{result?.totalRows?.toLocaleString() ?? 0}</span> baris
        {#if result?.executionMs !== undefined}
          <span style="color:var(--t3); margin-left:4px;">· {result.executionMs} ms</span>
        {/if}
      </span>

      <div style="flex:1;"></div>

      <span>Per halaman:</span>
      <select class="select select-sm" bind:value={size} onchange={() => { page = 1; run(); }} style="width:65px;">
        {#each [25,50,100,250] as n}<option value={n}>{n}</option>{/each}
      </select>

      <button class="btn btn-ghost btn-icon btn-xs" disabled={page <= 1}
        onclick={() => { page--; run(); }}><ChevronLeft size={14} /></button>
      <span class="mono" style="min-width:70px; text-align:center;">
        {page} / {Math.max(1, Math.ceil((result?.totalRows ?? 0) / size))}
      </span>
      <button class="btn btn-ghost btn-icon btn-xs"
        disabled={page >= Math.ceil((result?.totalRows ?? 0) / size)}
        onclick={() => { page++; run(); }}><ChevronRight size={14} /></button>
    </div>
  {/if}
</div>

{#if dialog === 'structure'}
  <TemplateDesigner
    source={cur}
    onClose={() => dialog = ''}
    onSaved={async (saved) => { dialog = ''; cur = saved; onChanged(saved); await reloadAll(); }}
  />
{:else if dialog === 'import'}
  <ImportModal tpl={cur} onClose={() => { dialog = ''; reloadAll(); }} onDone={reloadAll} />
{:else if dialog === 'export'}
  <ExportModal tpl={cur} onClose={() => dialog = ''} />
{:else if dialog === 'distinct'}
  <DistinctPanel tpl={cur} field={stepField} onPick={pickDistinct} onClose={() => dialog = ''} />
{:else if dialog === 'duplicate'}
  <DuplicatePanel tpl={cur} onApply={applyDuplicate} onClose={() => dialog = ''} />
{:else if dialog === 'row'}
  <RowEditor
    tpl={cur}
    rowId={editRowId}
    initial={editRowData}
    onClose={() => dialog = ''}
    onSaved={async () => { dialog = ''; await reloadAll(); }}
  />
{/if}
