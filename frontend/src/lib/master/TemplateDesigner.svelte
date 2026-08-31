<script lang="ts">
  import { showToast } from '../stores/appState';
  import { Plus, Trash2, X, Sparkles } from 'lucide-svelte';
  import {
    CreateTemplate, UpdateTemplate, SelectExcelFile, PreviewExcelFile
  } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let { source, onClose, onSaved }: {
    source: models.Template | null;               // null = workspace baru
    onClose: () => void;
    onSaved: (tpl: models.Template) => void;
  } = $props();

  const editing = !!source?.id;

  // A..Z, AA, AB, ... — cocokkan dengan ConvertIndexToExcelColumnName di backend
  const colLetter = (i: number) => {
    let out = '';
    for (let n = i; n >= 0; n = Math.floor(n / 26) - 1) out = String.fromCharCode(65 + (n % 26)) + out;
    return out;
  };

  const blank = (): models.Template => ({
    id:'', name:'', description:'', sheetName:'Sheet1',
    headerRow:1, dataStartRow:2, version:1, status:'ACTIVE',
    columns:[], recordCount:0, createdAt:new Date() as any, updatedAt:new Date() as any,
  } as unknown as models.Template);

  const newCol = (i: number) => ({
    id:'', templateId:t.id,
    excelColumn: colLetter(i),
    fieldName:`kolom_${i+1}`, displayName:`Kolom ${colLetter(i)}`,
    dataType:'STRING', formatPattern:'', required:false, isUnique:false,
    defaultValue:'', transformRules:JSON.stringify(['TRIM']),
    validationRules:'', sortOrder:i+1, isIndexed:true,
  }) as models.TemplateColumn;

  let t = $state<models.Template>(source ? JSON.parse(JSON.stringify(source)) : blank());
  if (!t.columns?.length) t.columns = [newCol(0)];

  let saving = $state(false), detecting = $state(false);

  const dtypes = [
    {v:'STRING',label:'Teks'}, {v:'INTEGER',label:'Angka Bulat'}, {v:'DECIMAL',label:'Desimal'},
    {v:'CURRENCY',label:'Mata Uang'}, {v:'PERCENTAGE',label:'Persentase'},
    {v:'DATE',label:'Tanggal'}, {v:'DATETIME',label:'Tanggal & Waktu'}, {v:'BOOLEAN',label:'Boolean'},
  ];
  const tforms = ['TRIM','UPPERCASE','LOWERCASE','CAPITALIZE','REMOVE_SPACE','NUMERIC_ONLY'];

  const getTF = (s: string) => { try { return JSON.parse(s||'[]'); } catch { return []; } };
  const setTF = (i: number, r: string) => {
    let l = getTF(t.columns[i].transformRules);
    l = l.includes(r) ? l.filter((x: string) => x !== r) : [...l, r];
    t.columns[i].transformRules = JSON.stringify(l);
  };

  async function autoDetect() {
    try {
      const fp = await SelectExcelFile(); if (!fp) return;
      detecting = true;
      const p = await PreviewExcelFile(fp, '', t.headerRow||1, 3);
      if (p?.headers?.length) {
        t.sheetName = p.activeSheet;
        t.columns = p.headers.map((h, i) => {
          const fn = h.toLowerCase().replace(/[^a-z0-9_]/g,'_').replace(/__+/g,'_').replace(/^_|_$/g,'') || `col_${i+1}`;
          return { ...newCol(i), fieldName: fn, displayName: h||`Kolom ${i+1}`, excelColumn: colLetter(i) };
        });
        showToast(`${p.headers.length} kolom terdeteksi`, 'success');
      }
    } catch (e: any) { showToast('Gagal baca Excel: ' + e, 'error'); }
    finally { detecting = false; }
  }

  async function save() {
    if (!t.name.trim()) { showToast('Nama workspace wajib diisi', 'warning'); return; }
    saving = true;
    try {
      const saved = editing ? await UpdateTemplate(t) : await CreateTemplate(t);
      showToast(editing ? 'Struktur diperbarui' : `Workspace "${saved.name}" dibuat`, 'success');
      onSaved(saved);
    } catch (e: any) { showToast('Gagal: ' + e, 'error'); }
    finally { saving = false; }
  }
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget) onClose(); }}>
  <div class="modal" style="width:920px; max-height:88vh;">
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">{editing ? `Struktur: ${source?.name}` : 'Workspace Baru'}</div>
        <div class="modal-hd-sub">
          {editing
            ? 'Ubah pemetaan kolom Excel → tabel SQLite workspace ini'
            : 'Langkah 1: beri nama workspace dan tentukan kolomnya'}
        </div>
      </div>
      <button class="btn btn-ghost btn-icon btn-xs" onclick={onClose}><X size={14} /></button>
    </div>

    <div class="modal-body" style="display:flex; flex-direction:column; gap:14px;">
      <div style="display:grid; grid-template-columns:1fr 160px 90px 90px auto; gap:10px; align-items:end;">
        <div class="field">
          <label class="field-label">Nama Workspace *</label>
          <input class="input" type="text" bind:value={t.name} placeholder="Data Peserta" />
        </div>
        <div class="field">
          <label class="field-label">Sheet Excel</label>
          <input class="input" type="text" bind:value={t.sheetName} placeholder="Sheet1" />
        </div>
        <div class="field">
          <label class="field-label">Baris Header</label>
          <input class="input" type="number" min="1" bind:value={t.headerRow} />
        </div>
        <div class="field">
          <label class="field-label">Mulai Data</label>
          <input class="input" type="number" min="2" bind:value={t.dataStartRow} />
        </div>
        <button class="btn btn-outline" onclick={autoDetect} disabled={detecting}>
          <Sparkles size={13} /> {detecting ? 'Membaca...' : 'Deteksi dari Excel'}
        </button>
      </div>

      <div class="field">
        <label class="field-label">Deskripsi</label>
        <input class="input" type="text" bind:value={t.description} placeholder="Keterangan workspace..." />
      </div>

      <hr class="sep" />

      <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:2px;">
        <span style="font-size:12.5px; font-weight:600; color:var(--t1);">Definisi Kolom</span>
        <button class="btn btn-outline btn-xs" onclick={() => t.columns = [...t.columns, newCol(t.columns.length)]}>
          <Plus size={12} /> Tambah Kolom
        </button>
      </div>

      <div class="inset" style="overflow:auto; max-height:320px;">
        <table class="tbl" style="min-width:700px;">
          <thead>
            <tr>
              <th style="width:58px;">Kolom</th>
              <th style="min-width:130px;">Field DB</th>
              <th style="min-width:140px;">Label UI</th>
              <th style="width:140px;">Tipe Data</th>
              <th style="width:50px; text-align:center;">Wajib</th>
              <th>Transformasi</th>
              <th style="width:32px;"></th>
            </tr>
          </thead>
          <tbody>
            {#each t.columns as col, i}
              <tr>
                <td>
                  <input class="input input-sm mono" type="text" bind:value={col.excelColumn}
                    style="width:46px; text-align:center; text-transform:uppercase; font-weight:700; color:var(--accent);" />
                </td>
                <td>
                  <input class="input input-sm mono" type="text" bind:value={col.fieldName} style="font-size:11.5px;" />
                </td>
                <td>
                  <input class="input input-sm" type="text" bind:value={col.displayName} />
                </td>
                <td>
                  <select class="select select-sm" bind:value={col.dataType}>
                    {#each dtypes as d}<option value={d.v}>{d.label}</option>{/each}
                  </select>
                </td>
                <td style="text-align:center;">
                  <input class="checkbox" type="checkbox" bind:checked={col.required} />
                </td>
                <td>
                  <div style="display:flex; flex-wrap:wrap; gap:3px;">
                    {#each tforms as r}
                      <button class="tag {getTF(col.transformRules).includes(r)?'on':''}" onclick={() => setTF(i,r)}>{r}</button>
                    {/each}
                  </div>
                </td>
                <td>
                  <button class="btn btn-ghost btn-icon btn-xs" style="color:var(--red);"
                    onclick={() => {
                      if (t.columns.length <= 1) { showToast('Minimal 1 kolom', 'warning'); return; }
                      t.columns = t.columns.filter((_,idx) => idx !== i);
                    }}>
                    <Trash2 size={12} />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <div class="modal-ft">
      <button class="btn btn-ghost" onclick={onClose}>Batal</button>
      <button class="btn btn-primary" onclick={save} disabled={saving}>
        {#if saving}<span class="spin"></span>{/if}
        {editing ? 'Simpan Perubahan' : 'Buat Workspace'}
      </button>
    </div>
  </div>
</div>
