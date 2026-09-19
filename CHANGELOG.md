# Changelog

Semua perubahan penting pada proyek **Natapadu (Navigasi Master Data dan Alat Terpadu)** akan didokumentasikan dalam berkas ini.

Format berkas ini mengikuti panduan [Keep a Changelog](https://keepachangelog.com/id/1.0.0/) dan menganut prinsip [Semantic Versioning](https://semver.org/).

---

## [0.0.4] - 2026-09-19

### 🚀 Fitur Baru
- **Dashboard Visualisasi Data Master**:
  - Halaman visualisasi data interaktif terintegrasi per workspace master data (`VisualizationView`).
  - Mendukung 6 jenis widget analitik:
    1. **Kartu Metrik (KPI Stat)**: Menghitung COUNT, COUNT DISTINCT, SUM, AVG, MIN, MAX.
    2. **Diagram Batang (Bar Chart)**: Membandingkan metrik antar kategori secara horizontal dengan persentase.
    3. **Grafik Tren Garis (Line Chart)**: Analisis tren deret waktu dengan titik data interaktif dan tooltip.
    4. **Bagan Donat (Donut Chart)**: Visualisasi proporsi dan sebaran pangsa kategori.
    5. **Peringkat Teratas (Top List / Leaderboard)**: Peringkat Top 10 dengan progress bar proporsional.
    6. **Tabel Ringkasan (Summary Table)**: Rangkuman nilai teragregasi per kelompok data.
  - **Filter Periode Global**:
    - Filter tanggal dinamis di sub-bar visualisasi: `Today`, `7 Days`, `30 Days`, `This Month`, `Last Month`, `All Time`, dan `Custom Range`.
    - Dropdown pemilihan kolom tanggal acuan (`Acuan:`) otomatis mendeteksi kolom `DATE`/`DATETIME` atau audit SQLite `_created_at`.
    - Perubahan periode langsung menyaring dan memperbarui seluruh widget secara serentak.
  - **Pengelompokan Tanggal Harian/Bulanan (Date Truncation)**:
    - Opsi grouping tanggal harian (`YYYY-MM-DD`), bulanan (`YYYY-MM`), atau tahunan (`YYYY`) pada query agregasi backend SQLite.
    - Menghilangkan jam/menit/detik sehingga data pendaftaran harian tanggal 1–30 dapat digabung menjadi satu titik grafik garis yang rapi.
  - **Penyimpanan Konfigurasi Widget**:
    - Konfigurasi widget disimpan otomatis ke database SQLite lokal dan langsung dipulihkan saat workspace dibuka kembali.

### 🎨 Tampilan & UI/UX Desktop Modern
- **Penyelarasan Desain dengan Halaman Workspace**:
  - Topbar visualisasi kini menggunakan tombol netral seragam (`btn btn-outline btn-xs`) dengan ukuran standar `30px` dan icon `size={12}`.
  - Kartu widget (`.widget-card`) menggunakan desain panel seamless (`var(--bg-3)`) tanpa header belang, selaras dengan gaya kartu workspace.
  - Tombol **Ubah** (Pencil) dan **Hapus** (Trash) dipindahkan ke bar footer kartu (`.widget-actions`), sama seperti aksi kartu di workspace list.
  - Empty state visualisasi yang bersih dengan icon `BarChart3`, deskripsi panduan, dan tombol kartu pilihan cepat (*Quick Pick Presets*).
  - Skema warna pill filter periode diselaraskan dengan tema resmi aplikasi (**royal blue solid `#2563eb`**).
  - Label teks `PERIOD:` dihilangkan agar antarmuka lebih bersih dan berfokus pada filter.

### 🛠️ Perbaikan Teknis & Backend
- Penambahan backend query `QueryVisualization`, `SaveVisualizationConfig`, dan `GetVisualizationConfig` pada Go service `DataGridService` dan Wails IPC bindings.
- Dukungan parsing filter tanggal `between` untuk kolom audit internal `_created_at` dan `_updated_at`.
- Penanganan agregasi dinamis SQL yang aman dan efisien langsung di level database SQLite.

---

## [0.0.3] - 2026-09-03
- Perbaikan layout CSS dan penyesuaian scroll area antarmuka dashboard.
- Desain desktop modern dark mode dengan border halus 1.5px.

## [0.0.2] - 2026-09-01
- Perbaikan build frontend dan build script CI/CD.
- Dukungan kemandirian artefak build `frontend/dist`.

## [0.0.1] - 2026-08-31
- Rilis inisial sistem Master Data Engine Natapadu (Wails v2 + Svelte 5 + SQLite).
- Import dan export Excel stream reader/writer.
- Server-side pagination dan dynamic SQL filter builder.
