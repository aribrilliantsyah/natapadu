<script lang="ts">
  import { onMount } from 'svelte';
  import { showToast } from '../stores/appState';
  import { AlertTriangle, X } from 'lucide-svelte';
  import { GetImportHistory, GetImportErrors, GetRecentLogs } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let tab = $state<'import'|'activity'>('import');
  let imports = $state<models.ImportHistory[]>([]);
  let acts = $state<models.ActivityLog[]>([]);
  let loading = $state(false);
  let errId = $state(''), errs = $state<models.ImportError[]>([]);
  let errOpen = $state(false), errLoading = $state(false);

  async function load() {
    loading = true;
    try {
      if (tab==='import') imports = await GetImportHistory('', 100);
      else acts = await GetRecentLogs(100);
    } catch { showToast('Gagal memuat log', 'error'); }
    finally { loading = false; }
  }

  async function openErr(id: string) {
    errId = id; errOpen = true; errLoading = true;
    try { errs = await GetImportErrors(id, 500); }
    catch { showToast('Gagal memuat detail error', 'error'); }
    finally { errLoading = false; }
  }

  onMount(load);
</script>

<div style="display:flex; flex-direction:column; height:100%;">
  <div class="topbar">
    <span class="topbar-title">Audit & Log</span>
    <div class="seg" style="margin-left:auto;">
      <button class="seg-btn {tab==='import'?'on':''}" onclick={() => { tab='import'; load(); }}>Riwayat Ingest</button>
      <button class="seg-btn {tab==='activity'?'on':''}" onclick={() => { tab='activity'; load(); }}>Log Aktivitas</button>
    </div>
  </div>

  <div style="flex:1; overflow:auto;">
    {#if tab === 'import'}
      <table class="tbl">
        <thead>
          <tr>
            <th>Status</th><th>File</th><th>Template</th>
            <th style="text-align:right;">Total</th>
            <th style="text-align:right; color:var(--green);">Sukses</th>
            <th style="text-align:right; color:var(--red);">Gagal</th>
            <th>Waktu</th><th>Operator</th><th></th>
          </tr>
        </thead>
        <tbody>
          {#if loading}
            <tr><td colspan={9} style="padding:50px; text-align:center; color:var(--t3);">Memuat...</td></tr>
          {:else if !imports.length}
            <tr><td colspan={9} style="padding:50px; text-align:center; color:var(--t3);">Belum ada riwayat import.</td></tr>
          {:else}
            {#each imports as j}
              <tr>
                <td><span class="badge {j.status==='COMPLETED'?'badge-green':'badge-red'}">{j.status}</span></td>
                <td class="truncate" style="max-width:180px;">{j.filename}</td>
                <td class="muted">{j.templateName||'—'}</td>
                <td class="mono" style="text-align:right;">{j.totalRows?.toLocaleString()}</td>
                <td class="mono" style="text-align:right; color:var(--green); font-weight:600;">{j.successRows?.toLocaleString()}</td>
                <td class="mono" style="text-align:right; color:var(--red);">{j.failedRows?.toLocaleString()}</td>
                <td class="mono" style="font-size:11.5px;">{new Date(j.startedAt).toLocaleString()}</td>
                <td class="muted">{j.importedBy}</td>
                <td>
                  {#if j.failedRows > 0}
                    <button class="btn btn-danger btn-xs" onclick={() => openErr(j.id)}>
                      <AlertTriangle size={11} /> Error Log
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    {:else}
      <table class="tbl">
        <thead>
          <tr><th>Waktu</th><th>User</th><th>Aksi</th><th>Target</th><th>Detail</th></tr>
        </thead>
        <tbody>
          {#if loading}
            <tr><td colspan={5} style="padding:50px; text-align:center; color:var(--t3);">Memuat...</td></tr>
          {:else if !acts.length}
            <tr><td colspan={5} style="padding:50px; text-align:center; color:var(--t3);">Belum ada log.</td></tr>
          {:else}
            {#each acts as log}
              <tr>
                <td class="mono" style="font-size:11.5px; white-space:nowrap;">{new Date(log.createdAt).toLocaleString()}</td>
                <td style="font-weight:500;">{log.username||'System'}</td>
                <td><span class="badge badge-blue">{log.action}</span></td>
                <td class="muted">{log.target||'—'}</td>
                <td class="muted truncate" style="max-width:280px;">{log.details}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    {/if}
  </div>
</div>

<!-- Error Modal -->
{#if errOpen}
  <div class="overlay" onclick={e => { if (e.target===e.currentTarget) errOpen=false; }}>
    <div class="modal" style="width:760px; max-height:78vh;">
      <div class="modal-hd">
        <div>
          <div class="modal-hd-title" style="display:flex; align-items:center; gap:7px;">
            <AlertTriangle size={15} style="color:var(--red);" /> Daftar Baris Ditolak
          </div>
          <div class="modal-hd-sub">Baris yang tidak lulus validasi tipe data / aturan field</div>
        </div>
        <button class="btn btn-ghost btn-icon btn-xs" onclick={() => errOpen=false}><X size={14} /></button>
      </div>
      <div class="modal-body" style="padding:0; overflow:auto;">
        {#if errLoading}
          <div class="empty"><div class="empty-sub">Memuat...</div></div>
        {:else if !errs.length}
          <div class="empty"><div class="empty-sub">Tidak ada error detail.</div></div>
        {:else}
          <table class="tbl">
            <thead>
              <tr>
                <th style="text-align:right;">Baris</th>
                <th>Kolom</th><th>Nilai Asli</th>
                <th style="color:var(--red);">Alasan</th>
              </tr>
            </thead>
            <tbody>
              {#each errs as e}
                <tr>
                  <td class="mono" style="text-align:right; color:var(--t3);">{e.rowNumber}</td>
                  <td style="font-weight:500;">{e.columnName}</td>
                  <td class="mono" style="font-size:12px; color:var(--t2); max-width:180px; overflow:hidden; text-overflow:ellipsis;">{e.fieldValue||'(kosong)'}</td>
                  <td style="color:var(--red);">{e.errorReason}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
      <div class="modal-ft">
        <button class="btn btn-outline" onclick={() => errOpen=false}>Tutup</button>
      </div>
    </div>
  </div>
{/if}
