<script lang="ts">
  import { onMount } from 'svelte';
  import { showToast } from '../stores/appState';
  import { Plus, Database, FileSpreadsheet, Download, Copy, Trash2, Layers, RefreshCw } from 'lucide-svelte';
  import {
    GetAllTemplates, DeleteTemplate, DuplicateTemplate, DownloadDataTemplate
  } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let { onOpen, onCreate }: {
    onOpen: (tpl: models.Template, mode?: 'data' | 'import') => void;
    onCreate: () => void;
  } = $props();

  let items = $state<models.Template[]>([]);
  let loading = $state(true);
  let busyId = $state('');

  export async function reload() {
    loading = true;
    try {
      const res = await GetAllTemplates();
      items = Array.isArray(res) ? res : [];
    } catch { showToast('Gagal memuat workspace', 'error'); items = []; }
    finally { loading = false; }
  }

  async function dlTemplate(tpl: models.Template) {
    busyId = tpl.id;
    try {
      const path = await DownloadDataTemplate(tpl.id, '');
      showToast(`Tersimpan: ${path}`, 'success', 'Template pengisian', 7000);
    } catch (e: any) { showToast('Gagal membuat template: ' + e, 'error'); }
    finally { busyId = ''; }
  }

  async function dup(tpl: models.Template) {
    try { await DuplicateTemplate(tpl.id, `${tpl.name} (Salinan)`); showToast('Workspace diduplikasi', 'success'); await reload(); }
    catch (e: any) { showToast('Gagal: ' + e, 'error'); }
  }

  async function del(tpl: models.Template) {
    if (!confirm(`Hapus workspace "${tpl.name}" beserta seluruh datanya? Tindakan ini tidak bisa dibatalkan.`)) return;
    try { await DeleteTemplate(tpl.id); showToast('Workspace dihapus', 'success'); await reload(); }
    catch (e: any) { showToast('Gagal hapus: ' + e, 'error'); }
  }

  const fmtDate = (d: any) => {
    try { return new Date(d).toLocaleDateString('id-ID', { day:'2-digit', month:'short', year:'numeric' }); }
    catch { return '—'; }
  };

  onMount(reload);
</script>

<div style="display:flex; flex-direction:column; height:100%;">
  <div class="topbar">
    <span class="topbar-title">Master Data</span>
    <button class="btn btn-ghost btn-icon btn-xs" title="Muat ulang" onclick={reload}>
      <RefreshCw size={13} style={loading ? 'animation:_spin 0.65s linear infinite;' : ''} />
    </button>
    <button class="btn btn-primary btn-xs" onclick={onCreate}>
      <Plus size={12} /> Workspace Baru
    </button>
  </div>

  <div class="scroll-area" style="padding:16px;">
    {#if loading}
      <div class="empty">
        <span class="spin" style="width:18px; height:18px; color:var(--accent);"></span>
        <div class="empty-sub" style="margin-top:4px;">Memuat workspace...</div>
      </div>

    {:else if items.length === 0}
      <!-- Placeholder: belum ada workspace sama sekali -->
      <div class="empty" style="padding:72px 24px;">
        <div style="
          width:60px; height:60px; border-radius:16px;
          background:var(--bg-3); border:1px solid var(--line-2);
          display:flex; align-items:center; justify-content:center;
          color:var(--t3); margin-bottom:10px;
        ">
          <Layers size={28} strokeWidth={1.5} />
        </div>
        <div class="empty-title">Belum Ada Workspace</div>
        <div class="empty-sub" style="max-width:400px;">
          Workspace menampung satu jenis master data — misalnya "Data Peserta".
          Dibuat sekali di awal, lalu bisa dibuka dan diisi ulang kapan saja.
        </div>
        <button class="btn btn-primary" style="margin-top:14px;" onclick={onCreate}>
          <Plus size={13} /> Buat Workspace Pertama
        </button>
      </div>

    {:else}
      <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(300px,1fr)); gap:12px;">
        {#each items as tpl (tpl.id)}
          <div class="panel ws-card">
            <div style="padding:14px 14px 0;">
              <div style="display:flex; align-items:flex-start; gap:8px;">
                <div style="flex:1; min-width:0;">
                  <div class="ws-name">{tpl.name}</div>
                  <div class="ws-desc">{tpl.description || 'Tanpa deskripsi'}</div>
                </div>
                {#if tpl.columns?.length}
                  <span class="badge badge-green">Siap</span>
                {:else}
                  <span class="badge badge-amber">Perlu setup</span>
                {/if}
              </div>

              <div style="display:flex; gap:16px; margin-top:12px;">
                <div>
                  <div class="ws-stat-num">{tpl.recordCount?.toLocaleString() ?? 0}</div>
                  <div class="ws-stat-lbl">Baris data</div>
                </div>
                <div>
                  <div class="ws-stat-num">{tpl.columns?.length ?? 0}</div>
                  <div class="ws-stat-lbl">Kolom</div>
                </div>
                <div>
                  <div class="ws-stat-num" style="font-size:13px; padding-top:3px;">{fmtDate(tpl.updatedAt)}</div>
                  <div class="ws-stat-lbl">Diperbarui</div>
                </div>
              </div>
            </div>

            <div class="ws-actions">
              <button class="btn btn-primary btn-xs" style="flex:1; justify-content:center;" onclick={() => onOpen(tpl)}>
                <Database size={12} /> Buka
              </button>
              <button class="btn btn-outline btn-xs" title="Langsung isi data" onclick={() => onOpen(tpl, 'import')}>
                <FileSpreadsheet size={12} /> Isi Data
              </button>
              <div style="flex:1;"></div>
              <button class="btn btn-ghost btn-icon btn-xs" title="Unduh template Excel kosong"
                disabled={busyId === tpl.id} onclick={() => dlTemplate(tpl)}>
                <Download size={13} />
              </button>
              <button class="btn btn-ghost btn-icon btn-xs" title="Duplikasi struktur" onclick={() => dup(tpl)}>
                <Copy size={13} />
              </button>
              <button class="btn btn-ghost btn-icon btn-xs" title="Hapus workspace"
                style="color:var(--red);" onclick={() => del(tpl)}>
                <Trash2 size={13} />
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .ws-card {
    display: flex;
    flex-direction: column;
    transition: border-color 120ms ease, transform 120ms ease;
  }
  .ws-card:hover { border-color: rgba(79,110,247,0.4); }
  .ws-name {
    font-size: 14px; font-weight: 700; color: var(--t1);
    letter-spacing: -0.2px;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .ws-desc {
    font-size: 11.5px; color: var(--t3); margin-top: 3px;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .ws-stat-num {
    font-size: 17px; font-weight: 700; color: var(--t1);
    font-variant-numeric: tabular-nums; line-height: 1.2;
  }
  .ws-stat-lbl {
    font-size: 10px; color: var(--t3); text-transform: uppercase;
    letter-spacing: 0.05em; font-weight: 600; margin-top: 2px;
  }
  .ws-actions {
    margin-top: 14px;
    padding: 10px 12px;
    border-top: 1px solid var(--line);
    display: flex; align-items: center; gap: 4px;
  }
</style>
