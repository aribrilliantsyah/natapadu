<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { showToast } from '../stores/appState';
  import { UploadCloud, FileSpreadsheet, XCircle, CheckCircle2, Eye, Download, X } from 'lucide-svelte';
  import {
    SelectExcelFile, PreviewExcelFile, StartImport, CancelImport, DownloadDataTemplate
  } from '../../../wailsjs/go/main/App';
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
  import type { models } from '../../../wailsjs/go/models';

  // Bentuk payload event 'import:progress' dari backend. Diketik lokal karena
  // Wails hanya men-generate tipe yang muncul di signature binding.
  type ImportProgress = {
    importId: string; processedRows: number; totalRows: number;
    successRows: number; failedRows: number; percent: number;
    speedRps: number; status: string; message: string;
  };

  let { tpl, onClose, onDone }: {
    tpl: models.Template;
    onClose: () => void;
    onDone: () => void;      // dipanggil setelah import sukses supaya grid dimuat ulang
  } = $props();

  let file = $state('');
  let preview = $state<models.ExcelSheetPreview | null>(null);
  let prevLoading = $state(false), importing = $state(false), importId = $state(''), dling = $state(false);
  let prog = $state<ImportProgress | null>(null);
  let result = $state<models.ImportHistory | null>(null);

  async function dlTemplate() {
    dling = true;
    try {
      const path = await DownloadDataTemplate(tpl.id, '');
      showToast(`Tersimpan: ${path}`, 'success', 'Template pengisian', 7000);
    } catch (e: any) { showToast('Gagal membuat template: ' + e, 'error'); }
    finally { dling = false; }
  }

  async function pickFile() {
    const p = await SelectExcelFile(); if (!p) return;
    file = p; result = null;
    prevLoading = true;
    try { preview = await PreviewExcelFile(p, tpl.sheetName, tpl.headerRow, 12); }
    catch (e: any) { showToast('Gagal preview: ' + e, 'error'); }
    finally { prevLoading = false; }
  }

  async function start() {
    if (!file) { showToast('Pilih file Excel dulu', 'warning'); return; }
    importing = true; result = null;
    prog = { importId:'', processedRows:0, totalRows:0, successRows:0, failedRows:0, percent:0, speedRps:0, status:'IN_PROGRESS', message:'Persiapan...' };
    try {
      result = await StartImport(tpl.id, file, '');
      showToast(`Import selesai: ${result.successRows.toLocaleString()} baris`, 'success');
      onDone();
    } catch (e: any) { showToast('Import gagal: ' + e, 'error'); }
    finally { importing = false; }
  }

  onMount(() => {
    EventsOn('import:progress', (d: ImportProgress) => { prog = d; importId = d.importId; });
  });
  onDestroy(() => EventsOff('import:progress'));
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget && !importing) onClose(); }}>
  <div class="modal" style="width:960px; height:620px; max-height:90vh;">
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">Isi Data — {tpl.name}</div>
        <div class="modal-hd-sub">Unduh template kosong, isi di Excel, lalu import ke workspace ini</div>
      </div>
      <button class="btn btn-ghost btn-icon btn-xs" disabled={importing} onclick={onClose}><X size={14} /></button>
    </div>

    <div style="flex:1; display:flex; overflow:hidden; min-height:0;">
      <!-- Kiri: konfigurasi -->
      <div class="pane-l" style="padding:14px; gap:12px; width:300px;">

        <button class="btn btn-outline" style="justify-content:flex-start;" disabled={dling} onclick={dlTemplate}>
          <Download size={13} /> {dling ? 'Menyiapkan...' : 'Unduh Template Excel Kosong'}
        </button>
        <div style="font-size:11px; color:var(--t3); margin-top:-6px; line-height:1.45;">
          Header sesuai struktur workspace + sheet petunjuk pengisian.
        </div>

        <hr class="sep" />

        <div class="field" style="flex:0;">
          <label class="field-label">File Sumber (.xlsx)</label>
          <button class="dropzone" onclick={pickFile}>
            <UploadCloud size={22} style="color:var(--t3);" />
            <span style="font-size:12.5px; font-weight:500;">Klik untuk memilih file</span>
            <span style="font-size:11px; color:var(--t3);">Mendukung file sangat besar</span>
          </button>
          {#if file}
            <div style="
              margin-top:6px; padding:7px 10px;
              background:var(--bg-3); border:1px solid var(--line); border-radius:7px;
              display:flex; align-items:center; gap:7px;
            ">
              <FileSpreadsheet size={13} style="color:var(--green); flex-shrink:0;" />
              <span style="font-size:12px; color:var(--t2); overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">
                {file.split('/').pop()?.split('\\').pop()}
              </span>
            </div>
          {/if}
        </div>

        <button class="btn btn-primary" style="width:100%; justify-content:center; height:32px;"
          onclick={start} disabled={importing || !file}>
          {#if importing}<span class="spin"></span>{/if}
          {importing ? 'Memproses...' : 'Mulai Import'}
        </button>

        {#if importing}
          <button class="btn btn-danger" style="width:100%; justify-content:center; height:28px;"
            onclick={() => CancelImport(importId)}>
            <XCircle size={13} /> Batalkan
          </button>
        {/if}

        {#if importing && prog}
          <div class="inset" style="padding:12px; display:flex; flex-direction:column; gap:8px;">
            <div style="display:flex; justify-content:space-between; font-size:11.5px; color:var(--t2);">
              <span>Batch Worker</span>
              <span class="mono" style="font-size:11px;">{prog.speedRps?.toLocaleString()||0} r/s</span>
            </div>
            <div class="prog-track"><div class="prog-fill anim"></div></div>
            <div style="display:grid; grid-template-columns:1fr 1fr; gap:6px; font-size:12px; text-align:center;">
              <div>
                <div style="color:var(--t3); font-size:10.5px; margin-bottom:2px;">Diproses</div>
                <div class="mono" style="font-weight:600;">{prog.processedRows?.toLocaleString()}</div>
              </div>
              <div>
                <div style="color:var(--green); font-size:10.5px; margin-bottom:2px;">Sukses</div>
                <div class="mono" style="font-weight:600; color:var(--green);">{prog.successRows?.toLocaleString()}</div>
              </div>
            </div>
          </div>
        {/if}

        {#if result}
          <div class="inset" style="padding:12px; display:flex; flex-direction:column; gap:8px; border-color:rgba(52,211,153,0.25);">
            <div style="display:flex; align-items:center; gap:6px; font-size:12.5px; font-weight:600; color:var(--green);">
              <CheckCircle2 size={14} /> Import Selesai
            </div>
            <div style="font-size:12px; display:flex; flex-direction:column; gap:3px;">
              <div style="display:flex; justify-content:space-between;">
                <span style="color:var(--t2);">Total</span>
                <span class="mono">{result.totalRows?.toLocaleString()}</span>
              </div>
              <div style="display:flex; justify-content:space-between;">
                <span style="color:var(--green);">Berhasil</span>
                <span class="mono" style="color:var(--green); font-weight:600;">{result.successRows?.toLocaleString()}</span>
              </div>
              <div style="display:flex; justify-content:space-between;">
                <span style="color:var(--red);">Gagal</span>
                <span class="mono" style="color:var(--red);">{result.failedRows?.toLocaleString()}</span>
              </div>
            </div>
            <button class="btn btn-outline" style="width:100%; justify-content:center; height:28px; font-size:12px;"
              onclick={onClose}>Lihat Datanya</button>
          </div>
        {/if}
      </div>

      <!-- Kanan: pratinjau -->
      <div class="pane-r">
        <div class="topbar" style="height:40px; background:var(--bg-3);">
          <Eye size={13} style="color:var(--t3);" />
          <span style="font-size:12.5px; font-weight:600; color:var(--t1); flex:1;">Pratinjau Sheet</span>
          {#if preview}
            <span class="badge badge-gray">{preview.activeSheet}</span>
            <span class="badge badge-gray">{preview.headers?.length} kolom</span>
          {/if}
        </div>
        <div style="flex:1; overflow:auto;">
          {#if prevLoading}
            <div class="empty"><div class="empty-sub">Membaca sampel data...</div></div>
          {:else if !preview?.headers?.length}
            <div class="empty">
              <div class="empty-sub">Pilih file Excel untuk melihat pratinjau header dan baris sampel data.</div>
            </div>
          {:else}
            <table class="tbl" style="min-width:max-content;">
              <thead>
                <tr>
                  <th style="color:var(--t3); width:36px; text-align:right;">#</th>
                  {#each preview.headers as h, i}
                    <th>
                      <span class="mono" style="color:var(--accent); margin-right:5px;">{String.fromCharCode(65+i)}</span>
                      {h}
                    </th>
                  {/each}
                </tr>
              </thead>
              <tbody>
                {#each preview.sampleRows as row, ri}
                  <tr>
                    <td class="mono" style="text-align:right; color:var(--t3);">{ri+1}</td>
                    {#each preview.headers as _, ci}
                      <td data-sel class="mono" style="font-size:12px;">{row[ci] ?? ''}</td>
                    {/each}
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .dropzone {
    width: 100%; padding: 20px 12px;
    border: 1.5px dashed var(--line-2);
    border-radius: 9px; background: var(--bg-3);
    cursor: pointer; color: var(--t2);
    display: flex; flex-direction: column; align-items: center; gap: 6px;
    transition: border-color 80ms, background 80ms;
    font-family: inherit;
  }
  .dropzone:hover { border-color: var(--accent); }
</style>
