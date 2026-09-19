<script lang="ts">
  import type { models } from '../../../wailsjs/go/models';
  import { X, Hash, BarChart3, TrendingUp, PieChart, Award, Table, Sparkles } from 'lucide-svelte';

  export interface WidgetConfig {
    id: string;
    title: string;
    type: 'card' | 'bar' | 'line' | 'donut' | 'toplist' | 'table';
    dimField: string;
    metricField: string;
    aggType: 'count' | 'distinct_count' | 'sum' | 'avg' | 'min' | 'max';
    dateTrunc?: 'day' | 'month' | 'year' | 'none';
    limit: number;
    sortBy: 'metric_desc' | 'metric_asc' | 'dim_asc' | 'dim_desc';
    width: '1' | '2' | '3';
    color: 'blue' | 'emerald' | 'violet' | 'amber' | 'rose' | 'cyan';
    unit: string;
  }

  let { tpl, widget, onSave, onClose }: {
    tpl: models.Template;
    widget: WidgetConfig | null;
    onSave: (w: WidgetConfig) => void;
    onClose: () => void;
  } = $props();

  const isEditing = !!widget;
  const cols = tpl.columns ?? [];

  // Nilai default
  const defaultDim = cols[0]?.fieldName ?? '';
  const defaultNumCol = cols.find(c =>
    ['INTEGER', 'DECIMAL', 'CURRENCY', 'PERCENTAGE'].includes((c.dataType || '').toUpperCase())
  )?.fieldName ?? '';

  let id = $state(widget?.id ?? `w_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`);
  let title = $state(widget?.title ?? '');
  let type = $state<WidgetConfig['type']>(widget?.type ?? 'bar');
  let dimField = $state(widget?.dimField ?? defaultDim);
  let metricField = $state(widget?.metricField ?? defaultNumCol);
  let aggType = $state<WidgetConfig['aggType']>(widget?.aggType ?? 'count');
  let dateTrunc = $state<WidgetConfig['dateTrunc']>(widget?.dateTrunc ?? (widget?.type === 'line' ? 'day' : 'day'));
  let limit = $state(widget?.limit ?? (widget?.type === 'line' ? 31 : 10));
  let sortBy = $state<WidgetConfig['sortBy']>(widget?.sortBy ?? (widget?.type === 'line' ? 'dim_asc' : 'metric_desc'));
  let width = $state<WidgetConfig['width']>(widget?.width ?? (widget?.type === 'card' ? '1' : '2'));
  let color = $state<WidgetConfig['color']>(widget?.color ?? 'blue');
  let unit = $state(widget?.unit ?? '');

  const needsMetricField = $derived(
    ['sum', 'avg', 'min', 'max'].includes(aggType)
  );

  const numericCols = $derived(
    cols.filter(c => ['INTEGER', 'DECIMAL', 'CURRENCY', 'PERCENTAGE'].includes((c.dataType || '').toUpperCase()))
  );

  const selectedCol = $derived(cols.find(c => c.fieldName === dimField));
  const isDateDimension = $derived(
    (selectedCol && (
      ['DATE', 'DATETIME'].includes((selectedCol.dataType || '').toUpperCase()) ||
      selectedCol.fieldName.toLowerCase().includes('tanggal') ||
      selectedCol.fieldName.toLowerCase().includes('tgl') ||
      selectedCol.fieldName.toLowerCase().includes('date') ||
      selectedCol.fieldName === '_created_at'
    )) || type === 'line'
  );

  const WIDGET_TYPES = [
    { id: 'card',    label: 'Kartu Metrik (KPI)', icon: Hash, desc: 'Satu angka ringkasan utama' },
    { id: 'bar',     label: 'Diagram Batang',     icon: BarChart3, desc: 'Perbandingan nilai antar kategori' },
    { id: 'line',    label: 'Grafik Garis',       icon: TrendingUp, desc: 'Visualisasi tren atau deret waktu' },
    { id: 'donut',   label: 'Bagan Donat',        icon: PieChart, desc: 'Proporsi & porsi persentase grup' },
    { id: 'toplist', label: 'Peringkat Teratas',  icon: Award, desc: 'Daftar Top 10 dengan bilah progress' },
    { id: 'table',   label: 'Tabel Ringkasan',    icon: Table, desc: 'Tabel ringkas nilai & kontribusi' },
  ] as const;

  const AGG_TYPES = [
    { id: 'count',          label: 'Jumlah Baris (COUNT)', desc: 'Hitung banyaknya baris data' },
    { id: 'distinct_count', label: 'Jumlah Unik (COUNT DISTINCT)', desc: 'Hitung nilai yang berbeda' },
    { id: 'sum',            label: 'Total Penjumlahan (SUM)', desc: 'Jumlahkan total nilai angka' },
    { id: 'avg',            label: 'Rata-rata (AVG)', desc: 'Nilai rata-rata dari kolom angka' },
    { id: 'max',            label: 'Nilai Tertinggi (MAX)', desc: 'Nilai angka paling besar' },
    { id: 'min',            label: 'Nilai Terendah (MIN)', desc: 'Nilai angka paling kecil' },
  ] as const;

  const COLOR_OPTIONS = [
    { id: 'blue',    name: 'Biru',    hex: '#3b82f6' },
    { id: 'emerald', name: 'Hijau',   hex: '#10b981' },
    { id: 'violet',  name: 'Ungu',    hex: '#8b5cf6' },
    { id: 'amber',   name: 'Kuning',  hex: '#f59e0b' },
    { id: 'rose',    name: 'Merah',   hex: '#f43f5e' },
    { id: 'cyan',    name: 'Cyan',    hex: '#06b6d4' },
  ] as const;

  function autoSuggestTitle() {
    const dimCol = cols.find(c => c.fieldName === dimField);
    const metCol = cols.find(c => c.fieldName === metricField);
    const dimName = dimCol?.displayName || dimField;
    const metName = metCol?.displayName || metricField;

    if (type === 'card') {
      if (aggType === 'count') return `Total Data ${tpl.name}`;
      if (aggType === 'distinct_count') return `Jumlah Unik ${dimName}`;
      if (aggType === 'sum') return `Total ${metName}`;
      if (aggType === 'avg') return `Rata-rata ${metName}`;
      if (aggType === 'max') return `${metName} Tertinggi`;
      if (aggType === 'min') return `${metName} Terendah`;
    }

    const metricText = needsMetricField ? metName : 'Jumlah';
    if (type === 'toplist') return `Top ${limit} ${dimName} (${metricText})`;
    if (type === 'bar') return `Grafik ${metricText} per ${dimName}`;
    if (type === 'donut') return `Distribusi ${dimName}`;
    if (type === 'line') {
      if (dateTrunc === 'day') return `Tren Harian ${metricText} (${dimName})`;
      if (dateTrunc === 'month') return `Tren Bulanan ${metricText} (${dimName})`;
      return `Tren ${metricText} per ${dimName}`;
    }
    if (type === 'table') return `Ringkasan per ${dimName}`;

    return `Visualisasi ${dimName}`;
  }

  function handleTypeChange(newType: WidgetConfig['type']) {
    type = newType;
    if (newType === 'card' && width === '2') {
      width = '1';
    } else if (newType !== 'card' && width === '1') {
      width = '2';
    }

    if (newType === 'line') {
      sortBy = 'dim_asc';
      limit = 31;
      dateTrunc = 'day';
    }
  }

  function submit() {
    const finalTitle = title.trim() || autoSuggestTitle();
    onSave({
      id,
      title: finalTitle,
      type,
      dimField,
      metricField: needsMetricField ? metricField : '',
      aggType,
      dateTrunc: dateTrunc || 'day',
      limit: Number(limit) || 10,
      sortBy,
      width,
      color,
      unit: unit.trim(),
    });
  }
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget) onClose(); }} role="presentation">
  <div class="modal" style="width:720px; max-height:90vh; display:flex; flex-direction:column;">
    <!-- Modal Header -->
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">{isEditing ? 'Ubah Widget Visualisasi' : 'Tambah Widget Visualisasi'}</div>
        <div class="modal-hd-sub">Pilih jenis grafik, kolom data, dan fungsi perhitungan untuk "{tpl.name}"</div>
      </div>
      <button type="button" class="btn btn-ghost btn-icon btn-xs" onclick={onClose} title="Tutup">
        <X size={14} />
      </button>
    </div>

    <!-- Modal Body -->
    <div class="modal-body" style="display:flex; flex-direction:column; gap:16px; overflow-y:auto;">
      <!-- Pilihan Tipe Visualisasi -->
      <div class="field">
        <span class="field-label">Tipe Visualisasi *</span>
        <div style="display:grid; grid-template-columns:repeat(3, 1fr); gap:8px; margin-top:2px;">
          {#each WIDGET_TYPES as wt}
            {@const Icon = wt.icon}
            <button
              type="button"
              class="type-card {type === wt.id ? 'active' : ''}"
              onclick={() => handleTypeChange(wt.id)}
            >
              <div class="type-card-title">
                <Icon size={16} />
                <span>{wt.label}</span>
              </div>
              <div class="type-card-desc">
                {wt.desc}
              </div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Judul Widget & Satuan -->
      <div style="display:grid; grid-template-columns:1fr 140px; gap:12px;">
        <div class="field">
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <label class="field-label" for="widget-title">Judul Widget</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              style="padding:0 6px; font-size:11px; color:#93c5fd;"
              onclick={() => title = autoSuggestTitle()}
            >
              <Sparkles size={11} style="margin-right:3px;" /> Otomatis
            </button>
          </div>
          <input
            id="widget-title"
            class="input"
            type="text"
            bind:value={title}
            placeholder={autoSuggestTitle()}
          />
        </div>

        <div class="field">
          <label class="field-label" for="widget-unit">Satuan / Suffix</label>
          <input
            id="widget-unit"
            class="input"
            type="text"
            bind:value={unit}
            placeholder="Rp, unit, dll."
          />
        </div>
      </div>

      <!-- Panel Sumber Data & Perhitungan -->
      <div class="inset" style="padding:14px; display:flex; flex-direction:column; gap:12px; background:var(--bg-3);">
        <div style="font-size:11.5px; font-weight:700; color:var(--t1); text-transform:uppercase; letter-spacing:0.05em;">
          Sumber Data & Perhitungan
        </div>

        <div style="display:grid; grid-template-columns:1fr 1fr; gap:12px;">
          <!-- Kolom Kategori / Dimensi -->
          <div class="field">
            <label class="field-label" for="dim-field">
              {type === 'card' && aggType !== 'distinct_count' ? 'Kolom Acuan (Opsional)' : 'Kolom Kategori / Sumbu X *'}
            </label>
            <select id="dim-field" class="select" bind:value={dimField}>
              {#each cols as c}
                <option value={c.fieldName}>
                  {c.displayName || c.fieldName} ({c.dataType || 'STRING'})
                </option>
              {/each}
            </select>
          </div>

          <!-- Fungsi Perhitungan (Agregasi) -->
          <div class="field">
            <label class="field-label" for="agg-type">Fungsi Perhitungan *</label>
            <select id="agg-type" class="select" bind:value={aggType}>
              {#each AGG_TYPES as a}
                <option value={a.id}>{a.label}</option>
              {/each}
            </select>
          </div>
        </div>

        <!-- Kolom Numerik jika fungsi butuh angka -->
        {#if needsMetricField}
          <div class="field" style="border-top:1px solid var(--line); padding-top:10px;">
            <label class="field-label" for="metric-field">
              Kolom Nilai Numerik (dihitung {aggType.toUpperCase()}) *
            </label>
            <select id="metric-field" class="select" bind:value={metricField}>
              {#if numericCols.length === 0}
                <option value="" disabled>Tidak ada kolom bertipe numerik</option>
              {/if}
              {#each cols as c}
                <option value={c.fieldName}>
                  {c.displayName || c.fieldName} ({c.dataType || 'STRING'})
                </option>
              {/each}
            </select>
            {#if numericCols.length === 0}
              <div style="font-size:11.5px; color:var(--amber); margin-top:2px;">
                Catatan: Untuk fungsi SUM atau AVG, disarankan memilih kolom bertipe INTEGER, DECIMAL, atau CURRENCY.
              </div>
            {/if}
          </div>
        {/if}

        <!-- Pengelompokan Waktu / Tanggal jika kolom tanggal atau tipe grafik garis -->
        {#if isDateDimension}
          <div class="field" style="border-top:1px solid var(--line); padding-top:10px;">
            <label class="field-label" for="date-trunc">
              Pengelompokan Waktu / Tanggal (Grouping)
            </label>
            <select id="date-trunc" class="select" bind:value={dateTrunc}>
              <option value="day">Per Hari / Tanggal (YYYY-MM-DD — Cocok untuk grafik bulanan tgl 1 - 30)</option>
              <option value="month">Per Bulan (YYYY-MM — Tren pergerakan bulanan)</option>
              <option value="year">Per Tahun (YYYY — Tren tahunan)</option>
              <option value="none">Nilai Asli Kolom (Tanpa pengelompokan tanggal)</option>
            </select>
            <div style="font-size:11px; color:var(--t3); margin-top:3px;">
              Memotong jam/menit/detik sehingga seluruh baris pada tanggal yang sama digabung menjadi 1 titik data harian.
            </div>
          </div>
        {/if}

        <!-- Batas & Pengurutan untuk grafik grouped -->
        {#if type !== 'card'}
          <div style="display:grid; grid-template-columns:1fr 1fr; gap:12px; border-top:1px solid var(--line); padding-top:10px;">
            <div class="field">
              <label class="field-label" for="sort-by">Urutan Tampilan</label>
              <select id="sort-by" class="select" bind:value={sortBy}>
                {#if isDateDimension}
                  <option value="dim_asc">Kronologis Tanggal Awal ke Akhir (ASC — Rekomendasi)</option>
                  <option value="dim_desc">Kronologis Tanggal Akhir ke Awal (DESC)</option>
                {/if}
                <option value="metric_desc">Nilai Tertinggi ke Terendah (DESC)</option>
                <option value="metric_asc">Nilai Terendah ke Tertinggi (ASC)</option>
                {#if !isDateDimension}
                  <option value="dim_asc">Nama Kategori A - Z (ASC)</option>
                  <option value="dim_desc">Nama Kategori Z - A (DESC)</option>
                {/if}
              </select>
            </div>

            <div class="field">
              <label class="field-label" for="limit-count">Batas Jumlah Item / Titik</label>
              <select id="limit-count" class="select" bind:value={limit}>
                <option value={10}>10 Item</option>
                <option value={15}>15 Item</option>
                <option value={31}>31 Item (1 Bulan tgl 1 - 30/31)</option>
                <option value={60}>60 Item (2 Bulan)</option>
                <option value={90}>90 Item (3 Bulan / Triwulan)</option>
                <option value={180}>180 Item (Semester)</option>
                <option value={365}>365 Item (1 Tahun Penuh)</option>
                <option value={500}>Maksimal (500 Item)</option>
              </select>
            </div>
          </div>
        {/if}
      </div>

      <!-- Tampilan: Lebar Grid & Palet Warna -->
      <div style="display:grid; grid-template-columns:1fr 1fr; gap:12px;">
        <div class="field">
          <label class="field-label" for="widget-width">Lebar Widget pada Grid</label>
          <select id="widget-width" class="select" bind:value={width}>
            <option value="1">1 Kolom (Kecil / Standar)</option>
            <option value="2">2 Kolom (Sedang / Lebar)</option>
            <option value="3">3 Kolom (Satu Baris Penuh)</option>
          </select>
        </div>

        <div class="field">
          <span class="field-label">Aksen Warna</span>
          <div style="display:flex; gap:8px; align-items:center; height:36px;">
            {#each COLOR_OPTIONS as co}
              <button
                type="button"
                class="color-dot {color === co.id ? 'active' : ''}"
                style="background:{co.hex};"
                title={co.name}
                onclick={() => color = co.id}
              ></button>
            {/each}
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Footer -->
    <div class="modal-ft">
      <button type="button" class="btn btn-ghost" onclick={onClose}>Batal</button>
      <button type="button" class="btn btn-primary" onclick={submit}>
        {isEditing ? 'Simpan Perubahan' : 'Pasang Widget'}
      </button>
    </div>
  </div>
</div>

<style>
  .type-card {
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
    text-align: left;
    transition: all 120ms ease;
    outline: none;
  }
  .type-card:hover:not(.active) {
    color: var(--t1);
    background: var(--bg-4);
    border-color: var(--line-2);
  }
  .type-card.active {
    background: var(--accent-dim);
    border-color: #2563eb;
    color: var(--t1);
    box-shadow: 0 0 0 1px #2563eb;
  }
  .type-card-title {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: 12.5px;
    font-weight: 600;
  }
  .type-card.active .type-card-title {
    color: #93c5fd;
  }
  .type-card-desc {
    font-size: 11px;
    color: var(--t3);
    line-height: 1.35;
  }
  .type-card.active .type-card-desc {
    color: var(--t2);
  }

  .color-dot {
    width: 26px;
    height: 26px;
    border-radius: 7px;
    border: 2px solid transparent;
    cursor: pointer;
    outline: none;
    transition: transform 120ms ease, border-color 120ms ease;
    box-sizing: border-box;
  }
  .color-dot:hover {
    transform: scale(1.1);
  }
  .color-dot.active {
    border-color: #ffffff;
    transform: scale(1.14);
  }
</style>
