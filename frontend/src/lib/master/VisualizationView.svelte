<script lang="ts">
  import { onMount } from 'svelte';
  import type { models } from '../../../wailsjs/go/models';
  import {
    QueryVisualization,
    GetVisualizationConfig,
    SaveVisualizationConfig
  } from '../../../wailsjs/go/main/App';
  import { showToast } from '../stores/appState';
  import {
    ArrowLeft, Plus, RefreshCw, Pencil, Trash2, ChevronLeft, ChevronRight,
    BarChart3, Hash, TrendingUp, PieChart, Award, Table, Sparkles,
    AlertCircle, Calendar as CalendarIcon, Clock
  } from 'lucide-svelte';
  import WidgetModal, { type WidgetConfig } from './WidgetModal.svelte';

  let { tpl, onBack }: {
    tpl: models.Template;
    onBack: () => void;
  } = $props();

  let widgets = $state<WidgetConfig[]>([]);
  let widgetData = $state<Record<string, models.VisualizationQueryResult>>({});
  let loadingWidgets = $state<Record<string, boolean>>({});
  let widgetErrors = $state<Record<string, string>>({});
  let initializing = $state(true);
  let refreshingAll = $state(false);

  let showModal = $state(false);
  let editingWidget = $state<WidgetConfig | null>(null);

  // ── FILTER PERIODE GLOBAL ──
  const PERIOD_OPTIONS = [
    { id: 'today',      label: 'Today' },
    { id: '7days',      label: '7 Days' },
    { id: '30days',     label: '30 Days' },
    { id: 'this_month', label: 'This Month' },
    { id: 'last_month', label: 'Last Month' },
    { id: 'all_time',   label: 'All Time' },
    { id: 'custom',     label: 'Custom Range' },
  ] as const;

  let selectedPeriod = $state<string>('all_time');
  let selectedDateField = $state<string>('');
  let customStart = $state<string>('');
  let customEnd = $state<string>('');

  const dateColumns = $derived.by(() => {
    const cols = tpl.columns ?? [];
    const dates = cols.filter(c =>
      ['DATE', 'DATETIME'].includes((c.dataType || '').toUpperCase()) ||
      c.fieldName.toLowerCase().includes('tanggal') ||
      c.fieldName.toLowerCase().includes('tgl') ||
      c.fieldName.toLowerCase().includes('date')
    ).map(c => ({
      fieldName: c.fieldName,
      displayName: c.displayName || c.fieldName,
    }));

    // Selalu sertakan kolom waktu import SQLite bawaan sebagai opsi acuan
    dates.push({
      fieldName: '_created_at',
      displayName: 'Waktu Input Data (_created_at)',
    });

    return dates;
  });

  const THEMES: Record<string, { primary: string; light: string; dim: string; bg: string; border: string }> = {
    blue:    { primary: '#2563eb', light: '#60a5fa', dim: 'rgba(37,99,235,0.18)', bg: 'rgba(37,99,235,0.08)', border: 'rgba(37,99,235,0.3)' },
    emerald: { primary: '#059669', light: '#34d399', dim: 'rgba(5,150,105,0.18)', bg: 'rgba(5,150,105,0.08)', border: 'rgba(5,150,105,0.3)' },
    violet:  { primary: '#7c3aed', light: '#a78bfa', dim: 'rgba(124,58,237,0.18)', bg: 'rgba(124,58,237,0.08)', border: 'rgba(124,58,237,0.3)' },
    amber:   { primary: '#d97706', light: '#fbbf24', dim: 'rgba(217,119,6,0.18)', bg: 'rgba(217,119,6,0.08)', border: 'rgba(217,119,6,0.3)' },
    rose:    { primary: '#e11d48', light: '#fb7185', dim: 'rgba(225,29,72,0.18)', bg: 'rgba(225,29,72,0.08)', border: 'rgba(225,29,72,0.3)' },
    cyan:    { primary: '#0891b2', light: '#22d3ee', dim: 'rgba(8,145,178,0.18)', bg: 'rgba(8,145,178,0.08)', border: 'rgba(8,145,178,0.3)' },
  };

  const DONUT_COLORS = [
    '#3b82f6', '#10b981', '#8b5cf6', '#f59e0b', '#f43f5e',
    '#06b6d4', '#ec4899', '#6366f1', '#14b8a6', '#84cc16'
  ];

  function toYMD(d: Date): string {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${y}-${m}-${day}`;
  }

  function getPeriodRange(p: string): { start: string; end: string } | null {
    const now = new Date();
    if (p === 'today') {
      const s = toYMD(now);
      return { start: s, end: s };
    }
    if (p === '7days') {
      const past = new Date(now);
      past.setDate(past.getDate() - 6);
      return { start: toYMD(past), end: toYMD(now) };
    }
    if (p === '30days') {
      const past = new Date(now);
      past.setDate(past.getDate() - 29);
      return { start: toYMD(past), end: toYMD(now) };
    }
    if (p === 'this_month') {
      const start = new Date(now.getFullYear(), now.getMonth(), 1);
      const end = new Date(now.getFullYear(), now.getMonth() + 1, 0);
      return { start: toYMD(start), end: toYMD(end) };
    }
    if (p === 'last_month') {
      const start = new Date(now.getFullYear(), now.getMonth() - 1, 1);
      const end = new Date(now.getFullYear(), now.getMonth(), 0);
      return { start: toYMD(start), end: toYMD(end) };
    }
    if (p === 'custom') {
      if (customStart && customEnd) {
        return { start: customStart, end: customEnd };
      }
      return null;
    }
    return null;
  }

  function handlePeriodChange(pId: string) {
    selectedPeriod = pId;
    if (pId === 'custom' && (!customStart || !customEnd)) {
      const now = new Date();
      const start = new Date(now.getFullYear(), now.getMonth(), 1);
      customStart = toYMD(start);
      customEnd = toYMD(now);
    }
    refreshAll();
  }

  function handleCustomDateChange() {
    if (selectedPeriod === 'custom' && customStart && customEnd) {
      refreshAll();
    }
  }

  function handleDateFieldChange() {
    if (selectedPeriod !== 'all_time') {
      refreshAll();
    }
  }

  async function loadConfig() {
    initializing = true;
    try {
      const raw = await GetVisualizationConfig(tpl.id);
      if (raw && raw.trim() && raw !== '[]') {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) {
          widgets = parsed;
        }
      }
    } catch (e) {
      console.error('Gagal membaca konfigurasi visualisasi:', e);
    } finally {
      initializing = false;
    }
    if (widgets.length > 0) {
      refreshAll();
    }
  }

  async function saveConfig(newList: WidgetConfig[]) {
    widgets = newList;
    try {
      await SaveVisualizationConfig(tpl.id, JSON.stringify(newList));
    } catch (e: any) {
      showToast('Gagal menyimpan konfigurasi: ' + e, 'error');
    }
  }

  async function loadWidgetData(w: WidgetConfig) {
    loadingWidgets[w.id] = true;
    delete widgetErrors[w.id];

    let filters: models.FilterCondition[] = [];
    const range = getPeriodRange(selectedPeriod);
    const dateField = selectedDateField || dateColumns[0]?.fieldName || '_created_at';

    if (range && dateField) {
      filters.push({
        fieldName: dateField,
        operator: 'between',
        value: range.start,
        valueTo: range.end,
      } as any);
    }

    try {
      const res = await QueryVisualization({
        templateId: tpl.id,
        widgetType: w.type,
        dimField: w.dimField,
        metricField: w.metricField,
        aggType: w.aggType,
        dateTrunc: w.dateTrunc || (w.type === 'line' ? 'day' : ''),
        sortBy: w.sortBy,
        limit: w.limit,
        filters,
        filterLogic: 'AND',
      } as any);
      widgetData[w.id] = res;
    } catch (e: any) {
      widgetErrors[w.id] = String(e?.message || e);
    } finally {
      loadingWidgets[w.id] = false;
    }
  }

  async function refreshAll() {
    refreshingAll = true;
    try {
      await Promise.all(widgets.map(w => loadWidgetData(w)));
    } finally {
      refreshingAll = false;
    }
  }

  function handleSaveWidget(w: WidgetConfig) {
    const idx = widgets.findIndex(item => item.id === w.id);
    let updated: WidgetConfig[];
    if (idx >= 0) {
      updated = [...widgets];
      updated[idx] = w;
      showToast('Widget berhasil diperbarui', 'success');
    } else {
      updated = [...widgets, w];
      showToast('Widget berhasil ditambahkan', 'success');
    }
    showModal = false;
    editingWidget = null;
    saveConfig(updated);
    loadWidgetData(w);
  }

  function handleDeleteWidget(id: string) {
    const updated = widgets.filter(w => w.id !== id);
    saveConfig(updated);
    delete widgetData[id];
    delete loadingWidgets[id];
    delete widgetErrors[id];
    showToast('Widget dihapus', 'info');
  }

  function clearAllWidgets() {
    if (!confirm('Hapus semua widget pada dashboard visual ini?')) return;
    saveConfig([]);
    widgetData = {};
    showToast('Semua widget berhasil dibersihkan', 'info');
  }

  function moveWidget(index: number, direction: 'left' | 'right') {
    const target = direction === 'left' ? index - 1 : index + 1;
    if (target < 0 || target >= widgets.length) return;
    const updated = [...widgets];
    const temp = updated[index];
    updated[index] = updated[target];
    updated[target] = temp;
    saveConfig(updated);
  }

  function openAddModal(initialType?: WidgetConfig['type']) {
    if (initialType) {
      editingWidget = {
        id: `w_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
        title: '',
        type: initialType,
        dimField: tpl.columns?.[0]?.fieldName ?? '',
        metricField: '',
        aggType: initialType === 'card' ? 'count' : 'count',
        limit: 10,
        sortBy: 'metric_desc',
        width: initialType === 'card' ? '1' : '2',
        color: 'blue',
        unit: '',
      };
    } else {
      editingWidget = null;
    }
    showModal = true;
  }

  function openEditModal(w: WidgetConfig) {
    editingWidget = w;
    showModal = true;
  }

  function createPresetWidgets() {
    const cols = tpl.columns ?? [];
    if (cols.length === 0) return;

    const firstCol = cols[0].fieldName;
    const numCol = cols.find(c =>
      ['INTEGER', 'DECIMAL', 'CURRENCY', 'PERCENTAGE'].includes((c.dataType || '').toUpperCase())
    )?.fieldName || '';

    const newWidgets: WidgetConfig[] = [
      {
        id: `w_total_${Date.now()}`,
        title: `Total Baris Data`,
        type: 'card',
        dimField: firstCol,
        metricField: '',
        aggType: 'count',
        limit: 10,
        sortBy: 'metric_desc',
        width: '1',
        color: 'blue',
        unit: 'baris',
      },
      {
        id: `w_distinct_${Date.now()}`,
        title: `Jumlah Unik ${cols[0].displayName || firstCol}`,
        type: 'card',
        dimField: firstCol,
        metricField: '',
        aggType: 'distinct_count',
        limit: 10,
        sortBy: 'metric_desc',
        width: '1',
        color: 'emerald',
        unit: 'entitas',
      },
    ];

    if (numCol) {
      const numColName = cols.find(c => c.fieldName === numCol)?.displayName || numCol;
      newWidgets.push({
        id: `w_sum_${Date.now()}`,
        title: `Total ${numColName}`,
        type: 'card',
        dimField: firstCol,
        metricField: numCol,
        aggType: 'sum',
        limit: 10,
        sortBy: 'metric_desc',
        width: '1',
        color: 'amber',
        unit: '',
      });
    }

    newWidgets.push({
      id: `w_top_${Date.now()}`,
      title: `Top 10 ${cols[0].displayName || firstCol}`,
      type: 'bar',
      dimField: firstCol,
      metricField: '',
      aggType: 'count',
      limit: 10,
      sortBy: 'metric_desc',
      width: '2',
      color: 'violet',
      unit: '',
    });

    if (cols.length > 1) {
      const secondCol = cols[1].fieldName;
      newWidgets.push({
        id: `w_donut_${Date.now()}`,
        title: `Distribusi ${cols[1].displayName || secondCol}`,
        type: 'donut',
        dimField: secondCol,
        metricField: '',
        aggType: 'count',
        limit: 6,
        sortBy: 'metric_desc',
        width: '1',
        color: 'cyan',
        unit: '',
      });
    }

    saveConfig(newWidgets);
    refreshAll();
    showToast('Dashboard rekomendasi berhasil dibuat!', 'success');
  }

  function getWidgetTypeIcon(type: WidgetConfig['type']) {
    switch (type) {
      case 'card': return Hash;
      case 'bar': return BarChart3;
      case 'line': return TrendingUp;
      case 'donut': return PieChart;
      case 'toplist': return Award;
      case 'table': return Table;
      default: return BarChart3;
    }
  }

  onMount(() => {
    if (dateColumns.length > 0 && !selectedDateField) {
      selectedDateField = dateColumns[0].fieldName;
    }
    loadConfig();
  });
</script>

<div style="display:flex; flex-direction:column; height:100%; overflow:hidden; background:var(--bg);">
  <!-- Topbar Visualisasi — Selaras Penuh dengan Topbar Workspace / Data Master -->
  <div class="topbar">
    <button
      type="button"
      class="btn btn-ghost btn-icon btn-xs"
      title="Kembali ke tabel data"
      onclick={onBack}
    >
      <ArrowLeft size={15} />
    </button>

    <span class="topbar-title" style="flex:0 0 auto;">{tpl.name}</span>
    <span class="badge badge-gray">Visualisasi Data</span>
    <span class="badge badge-gray">{widgets.length} widget</span>

    <div style="flex:1;"></div>

    {#if widgets.length > 0}
      <button
        type="button"
        class="btn btn-ghost btn-xs"
        style="color:var(--red);"
        onclick={clearAllWidgets}
        title="Hapus semua widget"
      >
        <Trash2 size={12} />
        Bersihkan
      </button>

      <button
        type="button"
        class="btn btn-outline btn-xs"
        onclick={refreshAll}
        disabled={refreshingAll}
        title="Muat ulang seluruh grafik"
      >
        <RefreshCw size={12} class={refreshingAll ? 'animate-spin' : ''} />
        Segarkan
      </button>

      <button
        type="button"
        class="btn btn-outline btn-xs"
        onclick={createPresetWidgets}
        title="Pasang otomatis widget rekomendasi"
      >
        <Sparkles size={12} />
        Rekomendasi
      </button>
    {/if}

    <button
      type="button"
      class="btn btn-outline btn-xs"
      onclick={() => openAddModal()}
      title="Tambah widget baru"
    >
      <Plus size={12} />
      Tambah Widget
    </button>
  </div>

  <!-- Subbar Filter Periode Global (Fokus ke Pill Filter dengan Color Scheme Resmi) -->
  <div class="period-toolbar">
    <!-- Segmented Pill Bar Tanpa Label Teks "PERIOD:" -->
    <div class="period-bar">
      {#each PERIOD_OPTIONS as opt}
        <button
          type="button"
          class="period-btn"
          class:active={selectedPeriod === opt.id}
          onclick={() => handlePeriodChange(opt.id)}
        >
          {#if opt.id === 'custom'}
            <CalendarIcon size={12} style="margin-right:4px;" />
          {/if}
          <span>{opt.label}</span>
        </button>
      {/each}
    </div>

    <!-- Date Range Picker jika Custom Range Dipilih -->
    {#if selectedPeriod === 'custom'}
      <div style="display:flex; align-items:center; gap:6px; margin-left:4px;">
        <input
          class="input input-sm mono"
          type="date"
          bind:value={customStart}
          onchange={handleCustomDateChange}
          style="width:130px; height:28px; padding:0 8px;"
        />
        <span style="font-size:11px; color:var(--t3);">s/d</span>
        <input
          class="input input-sm mono"
          type="date"
          bind:value={customEnd}
          onchange={handleCustomDateChange}
          style="width:130px; height:28px; padding:0 8px;"
        />
      </div>
    {/if}

    <!-- Pemilih Kolom Tanggal Acuan -->
    <div style="display:flex; align-items:center; gap:6px; margin-left:auto;">
      <Clock size={12} style="color:var(--t3);" />
      <span style="font-size:11.5px; color:var(--t3);">Acuan:</span>
      <select
        class="select select-sm"
        bind:value={selectedDateField}
        onchange={handleDateFieldChange}
        style="width:auto; min-width:140px; height:28px; font-size:11.5px;"
      >
        {#each dateColumns as dc}
          <option value={dc.fieldName}>
            {dc.displayName}
          </option>
        {/each}
      </select>
    </div>
  </div>

  <!-- Body Content -->
  <div class="scroll-area" style="padding:16px;">
    {#if initializing}
      <div class="empty" style="flex:1; padding:60px 24px;">
        <span class="spin" style="width:20px; height:20px; color:var(--accent);"></span>
        <div class="empty-sub" style="margin-top:8px;">Menyiapkan dashboard visual...</div>
      </div>

    {:else if widgets.length === 0}
      <!-- State Kosong Sesuai Komponen & Style Workspace -->
      <div class="empty" style="flex:1; max-width:540px; margin:40px auto; padding:48px 24px;">
        <div style="
          width:56px; height:56px; border-radius:15px;
          background:var(--bg-3); border:1px solid var(--line-2);
          display:flex; align-items:center; justify-content:center;
          color:var(--t3); margin-bottom:10px;
        ">
          <BarChart3 size={26} strokeWidth={1.5} />
        </div>

        <div class="empty-title">Belum Ada Widget Visualisasi</div>
        <div class="empty-sub">
          Susun ringkasan data untuk workspace "{tpl.name}" dengan kartu metrik KPI, diagram batang, grafik tren, atau bagan donat.
        </div>

        <div style="display:flex; gap:8px; margin-top:14px; flex-wrap:wrap; justify-content:center;">
          <button type="button" class="btn btn-outline btn-xs" onclick={() => openAddModal()}>
            <Plus size={12} /> Tambah Widget Baru
          </button>
          <button type="button" class="btn btn-outline btn-xs" onclick={createPresetWidgets}>
            <Sparkles size={12} /> Pasang Rekomendasi Otomatis
          </button>
        </div>

        <!-- Quick Pick Presets Showcase -->
        <div style="margin-top:24px; width:100%; border-top:1px solid var(--line); padding-top:16px;">
          <div style="font-size:10.5px; font-weight:600; color:var(--t3); text-transform:uppercase; letter-spacing:0.06em; margin-bottom:10px;">
            Pilihan Jenis Widget
          </div>

          <div style="display:grid; grid-template-columns:repeat(3, 1fr); gap:8px; text-align:left;">
            <button
              type="button"
              class="quick-card"
              onclick={() => openAddModal('card')}
            >
              <div class="quick-card-icon" style="background:rgba(37,99,235,0.12); color:#60a5fa;">
                <Hash size={15} />
              </div>
              <div style="font-size:12px; font-weight:600; color:var(--t1);">Kartu Metrik KPI</div>
              <div style="font-size:10.5px; color:var(--t3); line-height:1.3;">Angka ringkasan total / rata-rata</div>
            </button>

            <button
              type="button"
              class="quick-card"
              onclick={() => openAddModal('bar')}
            >
              <div class="quick-card-icon" style="background:rgba(124,58,237,0.12); color:#a78bfa;">
                <BarChart3 size={15} />
              </div>
              <div style="font-size:12px; font-weight:600; color:var(--t1);">Diagram Batang</div>
              <div style="font-size:10.5px; color:var(--t3); line-height:1.3;">Perbandingan nilai antar kategori</div>
            </button>

            <button
              type="button"
              class="quick-card"
              onclick={() => openAddModal('donut')}
            >
              <div class="quick-card-icon" style="background:rgba(8,145,178,0.12); color:#22d3ee;">
                <PieChart size={15} />
              </div>
              <div style="font-size:12px; font-weight:600; color:var(--t1);">Bagan Donat</div>
              <div style="font-size:10.5px; color:var(--t3); line-height:1.3;">Porsi persentase sebaran grup</div>
            </button>

            <button
              type="button"
              class="quick-card"
              onclick={() => openAddModal('line')}
            >
              <div class="quick-card-icon" style="background:rgba(5,150,105,0.12); color:#34d399;">
                <TrendingUp size={15} />
              </div>
              <div style="font-size:12px; font-weight:600; color:var(--t1);">Grafik Tren Garis</div>
              <div style="font-size:10.5px; color:var(--t3); line-height:1.3;">Perkembangan data deret waktu</div>
            </button>

            <button
              type="button"
              class="quick-card"
              onclick={() => openAddModal('toplist')}
            >
              <div class="quick-card-icon" style="background:rgba(217,119,6,0.12); color:#fbbf24;">
                <Award size={15} />
              </div>
              <div style="font-size:12px; font-weight:600; color:var(--t1);">Peringkat Top 10</div>
              <div style="font-size:10.5px; color:var(--t3); line-height:1.3;">Urutan teratas dengan bar progress</div>
            </button>

            <button
              type="button"
              class="quick-card"
              onclick={() => openAddModal('table')}
            >
              <div class="quick-card-icon" style="background:rgba(225,29,72,0.12); color:#fb7185;">
                <Table size={15} />
              </div>
              <div style="font-size:12px; font-weight:600; color:var(--t1);">Tabel Ringkasan</div>
              <div style="font-size:10.5px; color:var(--t3); line-height:1.3;">Tabel data ringkas per kelompok</div>
            </button>
          </div>
        </div>
      </div>

    {:else}
      <!-- Grid Widget Visualisasi — Seamless & Compact seperti di Workspace -->
      <div style="
        display:grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: 12px;
      ">
        {#each widgets as w, i (w.id)}
          {@const Icon = getWidgetTypeIcon(w.type)}
          {@const theme = THEMES[w.color] || THEMES.blue}
          {@const data = widgetData[w.id]}
          {@const isLoading = loadingWidgets[w.id]}
          {@const error = widgetErrors[w.id]}
          {@const spanClass = w.width === '3' ? '3' : w.width === '2' ? '2' : '1'}

          <div class="panel widget-card" style="grid-column: span {spanClass};">
            <!-- Header Kartu: Judul & Subtitle yang Menyatu -->
            <div style="padding: 12px 14px 0;">
              <div style="display:flex; align-items:flex-start; justify-content:space-between; gap:8px;">
                <div style="display:flex; align-items:center; gap:8px; min-width:0;">
                  <div style="
                    width: 22px; height: 22px; border-radius: 6px;
                    background: {theme.bg}; border: 1px solid {theme.border};
                    display: flex; align-items: center; justify-content: center;
                    color: {theme.light}; flex-shrink:0;
                  ">
                    <Icon size={12} />
                  </div>
                  <div class="widget-name" title={w.title}>{w.title}</div>
                </div>

                <span class="badge badge-gray" style="font-size:10px; text-transform:uppercase;">
                  {w.aggType}
                </span>
              </div>

              <div class="widget-sub">
                {#if w.metricField}
                  <span>{w.metricField}</span>
                {/if}
                {#if w.dimField && w.type !== 'card'}
                  <span>• per {w.dimField}</span>
                {/if}
                {#if w.dateTrunc && w.dateTrunc !== 'none'}
                  <span>• {w.dateTrunc === 'day' ? 'Harian' : w.dateTrunc === 'month' ? 'Bulanan' : 'Tahunan'}</span>
                {/if}
              </div>
            </div>

            <!-- Body Kartu: Grafik & Data Kompak -->
            <div class="widget-body">
              {#if isLoading}
                <div style="display:flex; align-items:center; justify-content:center; height:100px; color:var(--t3); gap:8px;">
                  <span class="spin" style="width:16px; height:16px;"></span>
                  <span style="font-size:12px;">Memuat data...</span>
                </div>

              {:else if error}
                <div style="display:flex; flex-direction:column; align-items:center; justify-content:center; height:100px; color:var(--red); gap:4px; text-align:center;">
                  <AlertCircle size={18} />
                  <div style="font-size:12px; font-weight:500;">Gagal memuat visualisasi</div>
                  <div style="font-size:11px; color:var(--t3); max-width:240px;">{error}</div>
                  <button type="button" class="btn btn-outline btn-xs" style="margin-top:4px;" onclick={() => openEditModal(w)}>
                    Sesuaikan Kolom
                  </button>
                </div>

              {:else if !data}
                <div style="display:flex; align-items:center; justify-content:center; height:100px; color:var(--t3); font-size:12px;">
                  Tidak ada data.
                </div>

              {:else}
                <!-- 1. KARTU METRIK (KPI STAT) -->
                {#if w.type === 'card'}
                  <div style="display:flex; flex-direction:column; gap:2px; padding: 4px 0;">
                    <div style="display:flex; align-items:baseline; gap:6px;">
                      <span class="mono" style="font-size:26px; font-weight:700; color:{theme.light}; line-height:1.15;">
                        {data.formatted || '0'}
                      </span>
                      {#if w.unit}
                        <span style="font-size:12px; font-weight:500; color:var(--t2);">{w.unit}</span>
                      {/if}
                    </div>

                    <div style="font-size:11px; color:var(--t3); margin-top:4px; display:flex; justify-content:space-between; align-items:center;">
                      <span>
                        {#if w.aggType === 'count'}Jumlah Total Data
                        {:else if w.aggType === 'distinct_count'}Jumlah Nilai Unik ({w.dimField})
                        {:else if w.aggType === 'sum'}Total {w.metricField}
                        {:else if w.aggType === 'avg'}Rata-rata {w.metricField}
                        {:else if w.aggType === 'max'}Nilai Maksimal {w.metricField}
                        {:else if w.aggType === 'min'}Nilai Minimal {w.metricField}
                        {/if}
                      </span>
                    </div>
                  </div>

                <!-- 2. DIAGRAM BATANG (BAR CHART) -->
                {:else if w.type === 'bar'}
                  {#if !data.data || data.data.length === 0}
                    <div style="color:var(--t3); font-size:11.5px; text-align:center; padding:16px 0;">Belum ada data.</div>
                  {:else}
                    {@const maxVal = Math.max(...data.data.map(p => p.value), 1)}
                    <div style="display:flex; flex-direction:column; gap:8px; max-height:220px; overflow-y:auto; padding-right:2px;">
                      {#each data.data as pt}
                        {@const barWidth = Math.max((pt.value / maxVal) * 100, 2)}
                        <div style="display:flex; flex-direction:column; gap:2px;">
                          <div style="display:flex; justify-content:space-between; font-size:11.5px;">
                            <span style="color:var(--t1); font-weight:500; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; max-width:65%;" title={pt.label}>
                              {pt.label}
                            </span>
                            <span class="mono" style="color:var(--t2); font-weight:600; font-size:11px;">
                              {pt.formatted} {w.unit} <span style="font-size:10px; color:var(--t3); font-weight:400;">({pt.percent.toFixed(1)}%)</span>
                            </span>
                          </div>
                          <div style="height:6px; width:100%; background:var(--bg-4); border-radius:3px; overflow:hidden;">
                            <div style="
                              height:100%; width:{barWidth}%;
                              background: {theme.primary};
                              border-radius:3px;
                              transition: width 0.3s ease;
                            "></div>
                          </div>
                        </div>
                      {/each}
                    </div>
                  {/if}

                <!-- 3. GRAFIK GARIS (LINE CHART) -->
                {:else if w.type === 'line'}
                  {#if !data.data || data.data.length < 2}
                    <div style="color:var(--t3); font-size:11.5px; text-align:center; padding:16px 0;">
                      Memerlukan minimal 2 titik data untuk membentuk grafik tren.
                    </div>
                  {:else}
                    {@const pts = data.data}
                    {@const maxV = Math.max(...pts.map(p => p.value), 1)}
                    {@const minV = Math.min(...pts.map(p => p.value), 0)}
                    {@const range = maxV - minV || 1}
                    {@const svgW = 380}
                    {@const svgH = 110}
                    {@const pad = 20}
                    {@const graphW = svgW - pad * 2}
                    {@const graphH = svgH - pad * 2}

                    {@const coords = pts.map((p, idx) => {
                      const x = pad + (idx / (pts.length - 1)) * graphW;
                      const y = pad + graphH - ((p.value - minV) / range) * graphH;
                      return { x, y, pt: p };
                    })}
                    {@const linePath = coords.map((c, i) => `${i === 0 ? 'M' : 'L'} ${c.x} ${c.y}`).join(' ')}
                    {@const areaPath = `${linePath} L ${coords[coords.length - 1].x} ${pad + graphH} L ${coords[0].x} ${pad + graphH} Z`}

                    <div style="display:flex; flex-direction:column; gap:6px;">
                      <svg viewBox="0 0 {svgW} {svgH}" style="width:100%; height:auto; overflow:visible;">
                        <line x1={pad} y1={pad} x2={pad + graphW} y2={pad} stroke="var(--line)" stroke-dasharray="3 3" />
                        <line x1={pad} y1={pad + graphH / 2} x2={pad + graphW} y2={pad + graphH / 2} stroke="var(--line)" stroke-dasharray="3 3" />
                        <line x1={pad} y1={pad + graphH} x2={pad + graphW} y2={pad + graphH} stroke="var(--line)" />

                        <!-- Area Flat Fill -->
                        <path d={areaPath} fill="{theme.primary}" fill-opacity="0.10" />

                        <!-- Line Stroke Solid -->
                        <path d={linePath} fill="none" stroke="{theme.light}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />

                        <!-- Dots -->
                        {#each coords as c}
                          {@const dotR = pts.length > 35 ? 2.2 : pts.length > 20 ? 2.8 : 3.5}
                          <circle cx={c.x} cy={c.y} r={dotR} fill="{theme.light}" stroke="var(--bg-3)" stroke-width="1.5">
                            <title>{c.pt.label}: {c.pt.formatted} {w.unit}</title>
                          </circle>
                        {/each}
                      </svg>

                      <div style="display:flex; justify-content:space-between; font-size:10.5px; color:var(--t3);">
                        <span class="mono" style="max-width:30%; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">{pts[0].label}</span>
                        {#if pts.length > 2}
                          {@const midIdx = Math.floor(pts.length / 2)}
                          <span class="mono" style="max-width:30%; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; text-align:center;">{pts[midIdx].label}</span>
                        {/if}
                        <span class="mono" style="max-width:30%; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; text-align:right;">{pts[pts.length - 1].label}</span>
                      </div>
                    </div>
                  {/if}

                <!-- 4. BAGAN DONAT (DONUT CHART) -->
                {:else if w.type === 'donut'}
                  {#if !data.data || data.data.length === 0}
                    <div style="color:var(--t3); font-size:11.5px; text-align:center; padding:16px 0;">Belum ada data.</div>
                  {:else}
                    {@const totalSum = data.data.reduce((acc, p) => acc + p.value, 0) || 1}
                    {@const circumference = 2 * Math.PI * 36}

                    <div style="display:flex; align-items:center; gap:16px; flex-wrap:wrap; justify-content:center;">
                      <div style="position:relative; width:96px; height:96px; flex-shrink:0;">
                        <svg viewBox="0 0 100 100" style="transform: rotate(-90deg); width:100%; height:100%;">
                          <circle cx="50" cy="50" r="36" fill="transparent" stroke="var(--bg-4)" stroke-width="11" />
                          {#each data.data as item, idx}
                            {@const itemFrac = item.value / totalSum}
                            {@const prevFrac = data.data.slice(0, idx).reduce((a, b) => a + b.value, 0) / totalSum}
                            {@const strokeLength = itemFrac * circumference}
                            {@const strokeOffset = -prevFrac * circumference}
                            {@const sliceColor = DONUT_COLORS[idx % DONUT_COLORS.length]}
                            <circle
                              cx="50" cy="50" r="36"
                              fill="transparent"
                              stroke={sliceColor}
                              stroke-width="11"
                              stroke-dasharray="{strokeLength} {circumference}"
                              stroke-dashoffset={strokeOffset}
                            >
                              <title>{item.label}: {item.formatted} ({item.percent.toFixed(1)}%)</title>
                            </circle>
                          {/each}
                        </svg>
                        <div style="position:absolute; inset:0; display:flex; flex-direction:column; align-items:center; justify-content:center; pointer-events:none;">
                          <span class="mono" style="font-size:12px; font-weight:700; color:var(--t1);">{data.data.length}</span>
                          <span style="font-size:9.5px; color:var(--t3);">Grup</span>
                        </div>
                      </div>

                      <div style="flex:1; min-width:130px; display:flex; flex-direction:column; gap:5px; max-height:140px; overflow-y:auto;">
                        {#each data.data as item, idx}
                          {@const sliceColor = DONUT_COLORS[idx % DONUT_COLORS.length]}
                          <div style="display:flex; align-items:center; gap:6px; font-size:11px;">
                            <span style="width:8px; height:8px; border-radius:2px; background:{sliceColor}; flex-shrink:0;"></span>
                            <span style="flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; color:var(--t1);" title={item.label}>
                              {item.label}
                            </span>
                            <span class="mono" style="font-weight:600; color:var(--t2); font-size:10.5px;">
                              {item.percent.toFixed(1)}%
                            </span>
                          </div>
                        {/each}
                      </div>
                    </div>
                  {/if}

                <!-- 5. TOP LIST (LEADERBOARD) -->
                {:else if w.type === 'toplist'}
                  {#if !data.data || data.data.length === 0}
                    <div style="color:var(--t3); font-size:11.5px; text-align:center; padding:16px 0;">Belum ada data.</div>
                  {:else}
                    {@const maxRankVal = Math.max(...data.data.map(p => p.value), 1)}
                    <div style="display:flex; flex-direction:column; gap:5px; max-height:220px; overflow-y:auto; padding-right:2px;">
                      {#each data.data as item, idx}
                        {@const fillPct = Math.max((item.value / maxRankVal) * 100, 3)}
                        <div style="
                          position:relative;
                          display:flex;
                          align-items:center;
                          justify-content:space-between;
                          padding:6px 10px;
                          border-radius:6px;
                          background:var(--bg-4);
                          overflow:hidden;
                          border:1px solid var(--line);
                        ">
                          <div style="
                            position:absolute; left:0; top:0; bottom:0;
                            width:{fillPct}%;
                            background:{theme.dim};
                            pointer-events:none;
                            transition: width 0.3s ease;
                          "></div>

                          <div style="position:relative; display:flex; align-items:center; gap:8px; min-width:0;">
                            <span style="
                              font-size:10px; font-weight:700; width:18px; height:18px; border-radius:50%;
                              display:flex; align-items:center; justify-content:center; flex-shrink:0;
                              background: {idx === 0 ? '#f59e0b' : idx === 1 ? '#94a3b8' : idx === 2 ? '#b45309' : 'var(--bg-5)'};
                              color: {idx < 3 ? '#ffffff' : 'var(--t2)'};
                            ">
                              {idx + 1}
                            </span>
                            <span style="font-size:12px; font-weight:500; color:var(--t1); overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title={item.label}>
                              {item.label}
                            </span>
                          </div>

                          <div class="mono" style="position:relative; font-size:11.5px; font-weight:700; color:{theme.light};">
                            {item.formatted} {w.unit}
                          </div>
                        </div>
                      {/each}
                    </div>
                  {/if}

                <!-- 6. TABEL RINGKASAN (SUMMARY TABLE) -->
                {:else if w.type === 'table'}
                  {#if !data.data || data.data.length === 0}
                    <div style="color:var(--t3); font-size:11.5px; text-align:center; padding:16px 0;">Belum ada data.</div>
                  {:else}
                    <div style="max-height:220px; overflow-y:auto;">
                      <table class="tbl" style="font-size:11.5px; width:100%;">
                        <thead>
                          <tr>
                            <th style="width:32px; text-align:center;">#</th>
                            <th>{w.dimField || 'Kategori'}</th>
                            <th style="text-align:right;">Nilai</th>
                            <th style="width:70px; text-align:right;">Porsi</th>
                          </tr>
                        </thead>
                        <tbody>
                          {#each data.data as item, idx}
                            <tr>
                              <td class="mono" style="text-align:center; color:var(--t3);">{idx + 1}</td>
                              <td style="font-weight:500; color:var(--t1);">{item.label}</td>
                              <td class="mono" style="text-align:right; font-weight:600;">
                                {item.formatted} {w.unit}
                              </td>
                              <td class="mono" style="text-align:right; color:var(--t2);">
                                {item.percent.toFixed(1)}%
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {/if}
                {/if}
              {/if}
            </div>

            <!-- Footer Kartu: Navigasi, Edit, Hapus (Persis seperti ws-actions di workspace) -->
            <div class="widget-actions">
              {#if data?.executionMs !== undefined}
                <span class="mono" style="font-size:10px; color:var(--t3); padding-left:4px;">
                  {data.executionMs}ms
                </span>
              {/if}

              <div style="flex:1;"></div>

              {#if i > 0}
                <button
                  type="button"
                  class="btn btn-ghost btn-icon btn-xs"
                  title="Pindahkan ke kiri"
                  onclick={() => moveWidget(i, 'left')}
                >
                  <ChevronLeft size={12} />
                </button>
              {/if}

              {#if i < widgets.length - 1}
                <button
                  type="button"
                  class="btn btn-ghost btn-icon btn-xs"
                  title="Pindahkan ke kanan"
                  onclick={() => moveWidget(i, 'right')}
                >
                  <ChevronRight size={12} />
                </button>
              {/if}

              <button
                type="button"
                class="btn btn-ghost btn-icon btn-xs"
                title="Ubah konfigurasi widget"
                onclick={() => openEditModal(w)}
              >
                <Pencil size={12} />
              </button>

              <button
                type="button"
                class="btn btn-ghost btn-icon btn-xs"
                title="Hapus widget"
                onclick={() => handleDeleteWidget(w.id)}
                style="color:var(--red);"
              >
                <Trash2 size={12} />
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<!-- Modal Dialog Tambah / Ubah Widget -->
{#if showModal}
  <WidgetModal
    {tpl}
    widget={editingWidget}
    onSave={handleSaveWidget}
    onClose={() => { showModal = false; editingWidget = null; }}
  />
{/if}

<style>
  /* ── Filter Periode Global Sesuai Color Scheme Aplikasi ── */
  .period-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 7px 16px;
    border-bottom: 1px solid var(--line);
    background: var(--bg-1);
    flex-shrink: 0;
    flex-wrap: wrap;
  }

  .period-bar {
    display: inline-flex;
    align-items: center;
    background: var(--ov-1);
    border: 1px solid var(--ov-4);
    border-radius: 8px;
    padding: 2.5px;
    gap: 2px;
  }

  .period-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0 11px;
    height: 25px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    color: var(--t2);
    background: transparent;
    border: none;
    outline: none;
    cursor: pointer;
    white-space: nowrap;
    transition: all 120ms ease;
    font-family: inherit;
  }

  .period-btn:hover:not(.active) {
    color: var(--t1);
    background: var(--ov-2);
  }

  /* Aktif mengikuti skema warna resmi aplikasi (royal blue) */
  .period-btn.active {
    background: #2563eb;
    color: #ffffff;
    font-weight: 600;
    box-shadow: 0 1px 4px rgba(37, 99, 235, 0.4);
  }

  /* ── Kartu Widget Seamless & Compact (Persis ws-card di Workspace) ── */
  .widget-card {
    display: flex;
    flex-direction: column;
    border-radius: 10px;
    background: var(--bg-3);
    border: 1px solid var(--line);
    transition: border-color 120ms ease, transform 120ms ease;
    overflow: hidden;
  }
  .widget-card:hover {
    border-color: rgba(79, 110, 247, 0.4);
  }
  .widget-name {
    font-size: 13.5px;
    font-weight: 700;
    color: var(--t1);
    letter-spacing: -0.2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .widget-sub {
    font-size: 11px;
    color: var(--t3);
    margin-top: 3px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .widget-body {
    padding: 12px 14px;
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 100px;
  }
  .widget-actions {
    padding: 6px 10px;
    border-top: 1px solid var(--line);
    display: flex;
    align-items: center;
    gap: 3px;
    background: rgba(0, 0, 0, 0.12);
  }

  /* ── Kartu Pilihan Cepat di Empty State ── */
  .quick-card {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    padding: 10px 12px;
    border-radius: 9px;
    border: 1px solid var(--line);
    background: var(--bg-3);
    color: var(--t2);
    cursor: pointer;
    font-family: inherit;
    transition: all 120ms ease;
    outline: none;
  }
  .quick-card:hover {
    background: var(--bg-4);
    border-color: var(--line-2);
    transform: translateY(-1px);
    box-shadow: var(--shadow-sm);
  }
  .quick-card-icon {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 2px;
  }
</style>
