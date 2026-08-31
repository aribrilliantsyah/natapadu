<script lang="ts">
  import { onMount } from 'svelte';
  import { activeTab, showToast } from '../stores/appState';
  import {
    Database, RefreshCw, ArrowRight, CheckCircle2, AlertTriangle, HardDrive, Layers, Activity
  } from 'lucide-svelte';
  import { GetDashboardSummary } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';
  import BarChart from '../charts/BarChart.svelte';
  import TrendChart from '../charts/TrendChart.svelte';

  let s = $state<models.AppSummary | null>(null);
  let loading = $state(true);

  const fmt = (n: number | undefined) => (n ?? 0).toLocaleString('id-ID');

  // Rasio keberhasilan seluruh riwayat import — angka utama kualitas data
  const totalProcessed = $derived((s?.successRows ?? 0) + (s?.failedRows ?? 0));
  const successRate = $derived(totalProcessed === 0 ? null : ((s?.successRows ?? 0) / totalProcessed) * 100);

  async function load() {
    loading = true;
    try { s = await GetDashboardSummary(); }
    catch { showToast('Gagal memuat dashboard', 'error'); }
    finally { loading = false; }
  }
  onMount(load);
</script>

<div style="display:flex; flex-direction:column; height:100%;">
  <div class="topbar">
    <span class="topbar-title">Dashboard</span>
    <button class="btn btn-ghost btn-icon btn-xs" onclick={load} title="Muat ulang">
      <RefreshCw size={13} style={loading ? 'animation:_spin 0.65s linear infinite;' : ''} />
    </button>
    <button class="btn btn-primary btn-xs" onclick={() => activeTab.set('master')}>
      <Database size={12} /> Master Data
    </button>
  </div>

  <div class="scroll-area" style="padding:16px; display:flex; flex-direction:column; gap:14px;">

    <!-- Angka utama -->
    <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:10px;">
      <div class="stat">
        <div class="stat-label">Total Baris Data</div>
        <div class="stat-val" style="font-size:28px;">{fmt(s?.totalRecords)}</div>
        <div class="stat-sub">Di seluruh workspace</div>
      </div>
      <div class="stat">
        <div class="stat-label">Tingkat Keberhasilan</div>
        <div class="stat-val" style="font-size:28px; color:{successRate === null ? 'var(--t1)' : successRate >= 99 ? 'var(--green)' : successRate >= 90 ? 'var(--amber)' : 'var(--red)'};">
          {successRate === null ? '—' : successRate.toFixed(1) + '%'}
        </div>
        <div class="stat-sub">
          {#if totalProcessed > 0}
            {fmt(s?.failedRows)} baris ditolak dari {fmt(totalProcessed)}
          {:else}
            Belum ada import
          {/if}
        </div>
      </div>
      <div class="stat">
        <div class="stat-label">Workspace</div>
        <div class="stat-val" style="font-size:28px;">{fmt(s?.totalTemplates)}</div>
        <div class="stat-sub">{fmt(s?.totalImports)} kali import</div>
      </div>
      <div class="stat">
        <div class="stat-label">Ukuran Database</div>
        <div class="stat-val" style="font-size:22px;">{s?.databaseSize ?? '—'}</div>
        <div class="stat-sub">SQLite lokal · mode WAL</div>
      </div>
    </div>

    <!-- Grafik -->
    <div style="display:grid; grid-template-columns:1.4fr 1fr; gap:12px;">
      <div class="panel" style="padding:14px 16px 16px;">
        <div class="ch-hd">
          <Activity size={13} style="color:var(--t3);" />
          <span class="ch-title">Aktivitas Import — 14 Hari Terakhir</span>
        </div>
        <TrendChart data={s?.importTrend ?? []} />
      </div>

      <div class="panel" style="padding:14px 16px 16px;">
        <div class="ch-hd">
          <Layers size={13} style="color:var(--t3);" />
          <span class="ch-title">Baris per Workspace</span>
        </div>
        <BarChart data={s?.workspaceSizes ?? []} empty="Belum ada workspace berisi data." />
      </div>
    </div>

    <div style="display:grid; grid-template-columns:1fr 1fr; gap:12px;">
      <div class="panel" style="padding:14px 16px 16px;">
        <div class="ch-hd">
          <HardDrive size={13} style="color:var(--t3);" />
          <span class="ch-title">Aktivitas Sistem — 30 Hari</span>
        </div>
        <BarChart data={s?.activityBreakdown ?? []} unit="kejadian" color="var(--chart-alt)"
          empty="Belum ada aktivitas tercatat." />
      </div>

      <!-- Ingest terakhir: sekaligus tampilan tabel dari grafik tren di atas -->
      <div class="panel" style="overflow:hidden; display:flex; flex-direction:column;">
        <div style="padding:12px 14px 9px; display:flex; align-items:center; justify-content:space-between; border-bottom:1px solid var(--line);">
          <span class="ch-title">Ingest Terakhir</span>
          <button class="btn btn-ghost btn-xs" onclick={() => activeTab.set('history')}>
            Semua <ArrowRight size={11} />
          </button>
        </div>
        {#if !s?.recentImports?.length}
          <div class="empty" style="padding:32px;">
            <div class="empty-sub">Belum ada aktivitas import.</div>
          </div>
        {:else}
          <div style="overflow:auto;">
            <table class="tbl">
              <thead>
                <tr>
                  <th>File</th>
                  <th>Workspace</th>
                  <th style="text-align:right;">Sukses</th>
                  <th style="text-align:right;">Gagal</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {#each s.recentImports as job}
                  <tr>
                    <td class="truncate" style="max-width:150px;">{job.filename}</td>
                    <td class="muted truncate" style="max-width:110px;">{job.templateName || '—'}</td>
                    <td class="mono" style="text-align:right; color:var(--green);">{fmt(job.successRows)}</td>
                    <td class="mono" style="text-align:right; color:{job.failedRows ? 'var(--red)' : 'var(--t3)'};">{fmt(job.failedRows)}</td>
                    <td>
                      <span class="badge {job.status==='COMPLETED'?'badge-green':'badge-red'}" style="display:inline-flex; align-items:center; gap:3px;">
                        {#if job.status === 'COMPLETED'}
                          <CheckCircle2 size={10} />
                        {:else}
                          <AlertTriangle size={10} />
                        {/if}
                        {job.status}
                      </span>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>

    <!-- Log aktivitas -->
    <div class="panel" style="overflow:hidden;">
      <div style="padding:12px 14px 9px; border-bottom:1px solid var(--line);">
        <span class="ch-title">Log Aktivitas Terbaru</span>
      </div>
      {#if !s?.recentActivity?.length}
        <div class="empty" style="padding:28px;"><div class="empty-sub">Belum ada log.</div></div>
      {:else}
        <div style="max-height:230px; overflow-y:auto;">
          {#each s.recentActivity as log}
            <div style="padding:9px 14px; border-bottom:1px solid var(--line); display:flex; align-items:center; gap:10px;">
              <span class="badge badge-blue" style="font-size:10px; flex-shrink:0;">{log.action}</span>
              <span style="flex:1; min-width:0; font-size:12px; color:var(--t2); overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">
                {log.details}
              </span>
              <span style="font-size:11px; color:var(--t3); flex-shrink:0;">{log.username || 'System'}</span>
              <span class="mono" style="font-size:11px; color:var(--t3); flex-shrink:0;">
                {new Date(log.createdAt).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })}
              </span>
            </div>
          {/each}
        </div>
      {/if}
    </div>

  </div>
</div>

<style>
  .ch-hd { display: flex; align-items: center; gap: 7px; margin-bottom: 12px; }
  .ch-title { font-size: 12.5px; font-weight: 600; color: var(--t1); }
</style>
