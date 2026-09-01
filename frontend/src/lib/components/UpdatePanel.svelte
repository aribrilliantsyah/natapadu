<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { showToast } from '../stores/appState';
  import { RefreshCw, Download, CheckCircle2, ArrowUpCircle, ExternalLink, FolderOpen } from 'lucide-svelte';
  import {
    GetAppVersion, CheckForUpdate, DownloadUpdate, InstallUpdate, OpenUpdateFolder, OpenReleasePage
  } from '../../../wailsjs/go/main/App';
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
  import type { updater } from '../../../wailsjs/go/models';

  let version = $state('—');
  let info = $state<updater.Info | null>(null);
  let checking = $state(false), downloading = $state(false), installing = $state(false);
  let progress = $state({ downloaded: 0, total: 0 });
  let installed = $state<{ needsRestart: boolean; message: string } | null>(null);
  let lastError = $state('');

  const pct = $derived(progress.total > 0 ? Math.round((progress.downloaded / progress.total) * 100) : 0);
  const mb = (n: number) => (n / 1024 / 1024).toFixed(1) + ' MB';

  async function check(manual = false) {
    checking = true; lastError = '';
    try {
      info = await CheckForUpdate();
      if (manual && !info.available) showToast('Anda sudah memakai versi terbaru', 'success');
    } catch (e: any) {
      lastError = String(e);
      if (manual) showToast('Gagal memeriksa pembaruan: ' + e, 'error');
    } finally { checking = false; }
  }

  async function download() {
    if (!info?.downloadUrl) return;
    downloading = true; progress = { downloaded: 0, total: info.assetSize ?? 0 };
    try {
      await DownloadUpdate(info.downloadUrl, info.assetName);
      showToast('Berkas pembaruan selesai diunduh', 'success');
    } catch (e: any) {
      showToast('Unduhan gagal: ' + e, 'error');
      downloading = false;
      return;
    }
    downloading = false;
    await install();
  }

  async function install() {
    installing = true;
    try {
      const res = await InstallUpdate();
      installed = { needsRestart: res.needsRestart, message: res.message };
    } catch (e: any) {
      showToast('Pemasangan gagal: ' + e, 'error');
    } finally { installing = false; }
  }

  onMount(async () => {
    try { version = await GetAppVersion(); } catch {}
    EventsOn('update:progress', (p: { downloaded: number; total: number }) => { progress = p; });
    // Pemeriksaan otomatis saat dibuka — hanya memeriksa, tidak mengunduh apa pun
    check(false);
  });
  onDestroy(() => EventsOff('update:progress'));
</script>

<div class="panel" style="padding:16px; display:flex; flex-direction:column; gap:12px;">
  <div style="display:flex; align-items:center; gap:8px;">
    <span style="font-size:13px; font-weight:600; color:var(--t1);">Versi & Pembaruan</span>
    <span class="badge badge-gray mono">{version}</span>
    <div style="flex:1;"></div>
    <button class="btn btn-ghost btn-xs" onclick={() => check(true)} disabled={checking || downloading}>
      <RefreshCw size={12} style={checking ? 'animation:_spin 0.65s linear infinite;' : ''} />
      Periksa
    </button>
  </div>

  <hr class="sep" />

  {#if installed}
    <div class="inset" style="padding:12px; border-color:rgba(52,211,153,0.3); display:flex; gap:9px; align-items:flex-start;">
      <CheckCircle2 size={16} style="color:var(--green); flex-shrink:0; margin-top:1px;" />
      <div style="min-width:0;">
        <div style="font-size:12.5px; font-weight:600; color:var(--green); margin-bottom:4px;">
          {installed.needsRestart ? 'Pembaruan Terpasang' : 'Berkas Siap Dipasang'}
        </div>
        <div style="font-size:11.5px; color:var(--t2); line-height:1.6; white-space:pre-wrap;">{installed.message}</div>
        {#if !installed.needsRestart}
          <button class="btn btn-outline btn-xs" style="margin-top:9px;" onclick={() => OpenUpdateFolder()}>
            <FolderOpen size={12} /> Buka Folder
          </button>
        {/if}
      </div>
    </div>

  {:else if downloading || installing}
    <div style="display:flex; flex-direction:column; gap:7px;">
      <div style="display:flex; justify-content:space-between; font-size:11.5px; color:var(--t2);">
        <span>{installing ? 'Memasang...' : 'Mengunduh pembaruan...'}</span>
        <span class="mono">
          {#if progress.total > 0}{mb(progress.downloaded)} / {mb(progress.total)} · {pct}%{/if}
        </span>
      </div>
      <div class="prog-track">
        <div class="prog-fill {progress.total === 0 || installing ? 'anim' : ''}"
          style={progress.total > 0 && !installing ? `width:${pct}%` : ''}></div>
      </div>
    </div>

  {:else if info?.available}
    <div class="inset" style="padding:12px; border-color:rgba(79,110,247,0.35); display:flex; flex-direction:column; gap:9px;">
      <div style="display:flex; align-items:center; gap:8px;">
        <ArrowUpCircle size={16} style="color:var(--accent);" />
        <span style="font-size:12.5px; font-weight:600; color:var(--t1);">
          Versi {info.latestVersion} tersedia
        </span>
        {#if info.publishedAt}
          <span style="font-size:11px; color:var(--t3);">· {info.publishedAt}</span>
        {/if}
      </div>

      {#if info.releaseNotes}
        <div style="font-size:11.5px; color:var(--t2); line-height:1.6; max-height:110px; overflow-y:auto; white-space:pre-wrap;">{info.releaseNotes}</div>
      {/if}

      {#if info.note}
        <div style="font-size:11.5px; color:var(--amber); line-height:1.5;">{info.note}</div>
      {/if}

      <div style="display:flex; gap:6px; align-items:center;">
        {#if info.downloadUrl}
          <button class="btn btn-primary btn-xs" onclick={download}>
            <Download size={12} />
            Unduh & Pasang{info.assetSize ? ` (${mb(info.assetSize)})` : ''}
          </button>
        {/if}
        <button class="btn btn-ghost btn-xs" onclick={() => info && OpenReleasePage(info.releaseUrl)}>
          <ExternalLink size={12} /> Lihat di GitHub
        </button>
      </div>

      <div style="font-size:11px; color:var(--t3); line-height:1.5;">
        Data Anda tidak tersentuh — berkas database berada di luar aplikasi dan
        tetap terpakai oleh versi baru.
      </div>
    </div>

  {:else if lastError}
    <div style="font-size:11.5px; color:var(--t3); line-height:1.5;">
      Tidak bisa memeriksa pembaruan saat ini. {lastError}
    </div>

  {:else if info}
    <div style="display:flex; align-items:center; gap:7px; font-size:12px; color:var(--t2);">
      <CheckCircle2 size={14} style="color:var(--green);" />
      Anda memakai versi terbaru.
    </div>

  {:else}
    <div style="font-size:12px; color:var(--t3);">Memeriksa pembaruan...</div>
  {/if}
</div>
