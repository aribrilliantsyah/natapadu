# Natapadu (Navigasi Master Data dan Alat Terpadu)

Aplikasi Desktop Multiplatform Berkinerja Tinggi untuk Pengelolaan Master Data Skala Besar (Jutaan Baris) dari Excel dengan Template Dinamis dan SQLite Lokal.

---

## 🌟 Fitur Utama

- **🚀 Arsitektur Data Management Engine**: Dirancang generik, memperlakukan Excel sebagai salah satu I/O tanpa hardcode skema tabel.
- **🛠️ Dynamic Template Engine**:
  - Konfigurasi pemetaan kolom Excel (Sheet, Header Row, Data Start Row).
  - Tipe data lengkap: `STRING`, `INTEGER`, `DECIMAL`, `CURRENCY`, `PERCENTAGE`, `DATE`, `DATETIME`, `BOOLEAN`.
  - Transformasi otomatis: `TRIM`, `UPPERCASE`, `LOWERCASE`, `CAPITALIZE`, `REMOVE_SPACE`, `NUMERIC_ONLY`.
  - Validasi aturan: `Required`, `Min/Max Length`, `Regex`, `Unique`.
  - Deteksi otomatis struktur kolom dari file Excel.
- **⚡ Batch Streaming Ingestion (Import)**:
  - Ekstraksi streaming menggunakan Excelize & transaksi batch SQLite (2,500 - 5,000 rows/batch).
  - Throughput tinggi (10,000 - 25,000+ baris/detik).
  - Progress real-time via event bridge Wails.
  - Pencatatan log error detail per baris.
- **📊 Master Data Grid Viewer**:
  - Server-side pagination & virtualized loading (tidak membebani RAM/DOM browser).
  - Multi-condition Filter Builder dinamis (`equals`, `contains`, `between`, `is_empty`, dll).
  - Global searching dan column sorting instan langsung di SQLite.
  - Hapus baris tunggal, bulk delete, atau truncate dataset.
- **💾 Streaming XLSX Exporter**:
  - Ekspor hasil query/filter/selected rows langsung ke format `.xlsx` menggunakan `StreamWriter`.
  - Pilihan subset kolom kustom.
- **🔒 Keamanan & Auth Lokal**:
  - Sistem login lokal 100% offline.
  - Password di-hash menggunakan `bcrypt`.
  - Default superadmin: `admin` / `admin123`.
- **📦 Backup & Audit History**:
  - Pencadangan berkas database SQLite aktif.
  - Riwayat audit pekerjaan import dan log aktivitas sistem.

---

## 🏗️ Tech Stack

- **Desktop Shell**: [Wails v2](https://wails.io) (Go + WebKit / WebView2)
- **Backend**: Go (Pure Go SQLite `modernc.org/sqlite`, `github.com/xuri/excelize/v2`, `golang.org/x/crypto/bcrypt`)
- **Frontend**: Svelte 5 + TypeScript + Tailwind CSS (v4)
- **Icons**: Lucide Svelte (`@lucide/svelte`)

---

## 💻 Panduan Menjalankan & Build

### 1. Prasyarat Sistem
- **Go**: Versi 1.22 atau lebih baru
- **Node.js**: Versi 18 atau lebih baru & npm
- **Wails CLI**:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### 2. Menjalankan Mode Development
```bash
# Jalankan live reload aplikasi desktop
wails dev -tags webkit2_41
```

### 3. Membangun Binary Native (Produksi)
```bash
# Build untuk Linux
wails build -tags webkit2_41 -clean

# Hasil build akan berada di:
# build/bin/natapadu-app
```

### 4. Build Multiplatform (CI Matrix)
Workflow GitHub Actions telah disediakan di `.github/workflows/build.yml` untuk menghasilkan artefak otomatis:
- **Windows**: `natapadu-windows-amd64.exe`
- **Linux**: `natapadu-linux-amd64`
- **macOS**: `natapadu-darwin-universal`
