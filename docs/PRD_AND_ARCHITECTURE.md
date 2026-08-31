# PRD & Technical Architecture Document
# Natapadu (Navigasi Master Data dan Alat Terpadu)

**Versi:** 1.1  
**Platform:** Multiplatform Desktop (Windows, macOS, Linux)  
**Architecture:** Local-first / Embedded Database  
**Target Pengguna:** Internal / Operasional / Data Administrator  

---

## 1. Executive Summary & Ringkasan Produk

Natapadu adalah aplikasi desktop cross-platform berkinerja tinggi untuk mengelola **master data berukuran besar** (ratusan ribu hingga jutaan baris) dari berbagai format file tabular (dimulai dari Excel `.xlsx`/`.xls`, dengan arsitektur generik yang siap untuk CSV, JSON, API). Data di-ingest ke **SQLite lokal** agar proses pencarian, filtering multi-kondisi, sorting, dan paginasi berjalan sangat cepat (< 50ms) langsung di mesin lokal tanpa ketergantungan koneksi server atau aplikasi Microsoft Excel.

### 1.1 Filosofi Inti: Data Management Engine
Natapadu **bukan sekadar pembaca/viewer Excel**, melainkan **Data Management Engine** mandiri.
1. **Dynamic Schema per Template**: Struktur data ditentukan melalui konfigurasi Template (sheet, header, kolom, tipe data, validasi, transform) — tanpa hardcode jenis data di level kode.
2. **Dedicated Physical SQLite Table per Template**: Setiap template memiliki tabel database fisik tersendiri (`dataset_<template_id>`) dengan indeks otomatis pada field yang relevan. Hal ini memastikan performa search/filter/sort tetap optimal pada jutaan record (menghindari overhead EAV/JSON).
3. **Pure-Go Embedded Engine**: Menggunakan driver SQLite murni (`modernc.org/sqlite`) dan parser Excel (`excelize/v2`) murni dalam Go untuk memastikan kemudahan cross-compilation native (Windows `.exe`, macOS `.app`, Linux binary/AppImage) tanpa dependensi CGO yang rumit.
4. **Server-Side Data Grid Interaction**: Frontend Svelte tidak pernah memuat jutaan baris sekaligus ke memori DOM, melainkan hanya mengambil slice data paged (10 - 500 rows) sesuai query SQL yang dibangun secara dinamis di backend Go.

---

## 2. Arsitektur Sistem & Multiplatform Tech Stack

```
+-----------------------------------------------------------------------+
|                            FRONTEND (UI)                             |
|  - Svelte 5 + TypeScript + Tailwind CSS (v4)                          |
|  - Virtualized / Server-paginated Data Grid                           |
|  - Visual Filter Builder & Saved Filters                              |
|  - Template Designer & Mapping Preview                                |
|  - Real-time Import Progress via Wails Event Bus                      |
+-----------------------------------------------------------------------+
                                  │ ▲ (Wails RPC / Event Bridge)
                                  ▼ │
+-----------------------------------------------------------------------+
|                            BACKEND (Go)                               |
|  - App Controller (Wails Bindings)                                    |
|  - Template Engine (CRUD, DDL Generator, Sanitizer, Validator)       |
|  - Import Engine (Stream Reader, Batch Inserter, Row Validation)      |
|  - Query Engine (SQL Builder, Paging, Dynamic Filter & Multi-Sort)    |
|  - Export Engine (Stream Excelize XLSX Generator from SQL Cursor)     |
|  - Auth & Security Engine (Argon2id/Bcrypt Hash, Local Session)       |
|  - Activity Log & Backup/Restore Manager                              |
+-----------------------------------------------------------------------+
                                  │ ▲
                                  ▼ │
+-----------------------------------------------------------------------+
|                     EMBEDDED STORAGE (SQLite)                         |
|  - Schema Metadata: templates, template_columns, datasets             |
|  - Audit & History: import_history, import_errors, activity_logs      |
|  - Users & Config:  users, saved_filters, app_settings                |
|  - Dynamic Datasets: dataset_1, dataset_2, ... (Indexed Columns)      |
+-----------------------------------------------------------------------+
```

### 2.1 Pilihan Komponen Stack

| Komponen | Pilihan | Alasan Teknis |
|---|---|---|
| **Framework Desktop** | **Wails v2 (Go + WebKit/WebView2)** | Binary native, konsumsi RAM rendah (< 80MB idle), integrasi RPC mulus antara Go dan TypeScript. |
| **Backend Language** | **Go 1.22+** | Kecepatan eksekusi native, konkurensi goroutine untuk background streaming import/export, kemudahan cross-compile. |
| **Frontend Framework**| **Svelte + TypeScript + Tailwind CSS** | Kompilasi tanpa virtual DOM overhead, performa tinggi untuk reactive state, ukuran bundle frontend minimal (< 1MB). |
| **Database Engine** | **SQLite via `modernc.org/sqlite`** | 100% Pure Go (CGO-free), file database portable tunggal (`.db`), support DDL dinamis, WAL mode, transaction batching. |
| **Excel Parser/Writer**| **`github.com/xuri/excelize/v2`** | Standard industri Go untuk XLSX, mendukung stream reading dan stream writing untuk meminimalkan footprint memori. |
| **Password Security** | **Argon2id / Bcrypt** | Standar keamanan modern untuk hashing password pengguna lokal. |

---

## 3. Database Schema & ERD (Entity Relationship Diagram)

### 3.1 Metadata & System Tables

```sql
-- 1. Tabel Pengguna Lokal
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'USER', -- 'ADMIN', 'USER'
    status TEXT NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'INACTIVE'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 2. Metadata Template Master Data
CREATE TABLE IF NOT EXISTS templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    sheet_name TEXT NOT NULL DEFAULT 'Sheet1',
    header_row INTEGER NOT NULL DEFAULT 1,
    data_start_row INTEGER NOT NULL DEFAULT 2,
    version INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'ARCHIVED'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 3. Definisi Kolom Template
CREATE TABLE IF NOT EXISTS template_columns (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    excel_column TEXT NOT NULL,          -- e.g. "A", "B", "AA"
    field_name TEXT NOT NULL,            -- e.g. "nik", "nama_lengkap" (Sanitized SQL identifier)
    display_name TEXT NOT NULL,          -- e.g. "Nomor Induk Kependudukan"
    data_type TEXT NOT NULL,             -- 'STRING', 'INTEGER', 'DECIMAL', 'BOOLEAN', 'DATE', 'DATETIME', 'CURRENCY', 'PERCENTAGE'
    format_pattern TEXT,                 -- e.g. "DD/MM/YYYY", "YYYY-MM-DD", "RP"
    required INTEGER NOT NULL DEFAULT 0, -- 1: Yes, 0: No
    is_unique INTEGER NOT NULL DEFAULT 0,-- 1: Yes, 0: No
    default_value TEXT,
    transform_rules TEXT,                -- JSON Array: ["TRIM", "UPPERCASE", "REMOVE_SPACE"]
    validation_rules TEXT,               -- JSON Object: {"min_length": 16, "max_length": 16, "regex": "^[0-9]+$"}
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_indexed INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
);

-- 4. Dataset Registry
CREATE TABLE IF NOT EXISTS datasets (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL UNIQUE,
    table_name TEXT NOT NULL UNIQUE,     -- e.g. "dataset_tpl_01h8..."
    record_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
);

-- 5. Import History Audit
CREATE TABLE IF NOT EXISTS import_history (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    file_size_bytes INTEGER NOT NULL DEFAULT 0,
    total_rows INTEGER NOT NULL DEFAULT 0,
    success_rows INTEGER NOT NULL DEFAULT 0,
    failed_rows INTEGER NOT NULL DEFAULT 0,
    imported_by TEXT NOT NULL,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME,
    status TEXT NOT NULL,                -- 'IN_PROGRESS', 'COMPLETED', 'FAILED', 'CANCELLED'
    error_message TEXT,
    FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
);

-- 6. Detail Error Import per Baris
CREATE TABLE IF NOT EXISTS import_errors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    import_id TEXT NOT NULL,
    row_number INTEGER NOT NULL,
    column_name TEXT,
    field_value TEXT,
    error_reason TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(import_id) REFERENCES import_history(id) ON DELETE CASCADE
);

-- 7. Saved Filters
CREATE TABLE IF NOT EXISTS saved_filters (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    name TEXT NOT NULL,
    filter_payload TEXT NOT NULL,        -- JSON format filter conditions
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
);

-- 8. Activity Logs
CREATE TABLE IF NOT EXISTS activity_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT,
    username TEXT,
    action TEXT NOT NULL,                -- 'LOGIN', 'IMPORT', 'EXPORT', 'CREATE_TEMPLATE', etc.
    target TEXT,
    details TEXT,                        -- JSON or summary text
    ip_address TEXT DEFAULT 'localhost',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 9. App Settings
CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 3.2 Dynamic Physical Table Pattern (`dataset_<template_id>`)
Setiap kali template dibuat atau dimodifikasi, backend akan men-generate tabel fisik yang di-sanitize:
```sql
CREATE TABLE IF NOT EXISTS dataset_<sanitized_template_id> (
    _row_id INTEGER PRIMARY KEY AUTOINCREMENT,
    _import_id TEXT NOT NULL,
    _created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    _updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    [field_1] [SQLITE_TYPE],
    [field_2] [SQLITE_TYPE],
    ...
);
-- Indeks otomatis pada kolom yang sering dicari/difilter
CREATE INDEX IF NOT EXISTS idx_<tbl>_[field_1] ON dataset_<sanitized_template_id> ([field_1]);
```

---

## 4. Spesifikasi Modul Teknis

### 4.1 Template Engine
- **Field Name Sanitizer**: Mengubah display name menjadi safe snake_case SQL identifier (`[a-z0-9_]`).
- **Data Type Mapping**:
  - `STRING` -> `TEXT`
  - `INTEGER` -> `INTEGER`
  - `DECIMAL` / `CURRENCY` / `PERCENTAGE` -> `REAL`
  - `BOOLEAN` -> `INTEGER` (0 / 1)
  - `DATE` / `DATETIME` -> `TEXT` (ISO-8601 `YYYY-MM-DD` / `YYYY-MM-DD HH:MM:SS`)
- **Transform Rules Pipeline**:
  - `TRIM`: Hapus spasi di awal/akhir
  - `UPPERCASE` / `LOWERCASE` / `CAPITALIZE`
  - `REMOVE_SPACE`: Hapus semua spasi
  - `NUMERIC_ONLY`: Ekstrak digit angka saja (misal untuk NIK, HP, Rekening)
  - `DATE_FORMAT`: Konversi string tanggal aneka format (`DD/MM/YYYY`, `DD-MM-YYYY`, Excel serial number) ke format standar SQLite (`YYYY-MM-DD`)
  - `CURRENCY_CLEAN`: Konversi `Rp 1.500.000,00` -> `1500000`
- **Validation Pipeline**:
  - `Required`: Not null or empty
  - `MinLength` / `MaxLength` / `ExactLength`
  - `Regex`: Pola regex kustom
  - `Unique`: Cek duplikasi di batch dan di tabel dataset SQLite

### 4.2 Import Engine & Batch Streaming
- **Excel Row Streaming**: Menggunakan `excelize.Rows()` iterator untuk memproses jutaan baris tanpa me-load seluruh worksheet ke RAM.
- **Worker & Batch Transactions**: Menggunakan transaksi SQLite per chunk (mis. 2,000 - 5,000 baris per `BEGIN TRANSACTION ... COMMIT`) untuk throughput tinggi (10,000 - 25,000+ baris/detik).
- **Real-time Event Emission**: Go memancarkan event `import:progress` (`{import_id, processed_rows, total_rows, valid_rows, invalid_rows, percent, speed_rps}`) via Wails Events ke frontend setiap 100ms.
- **Fail-safe Error Collector**: Baris yang gagal validasi dicatat ke `import_errors` dan dapat di-download ulang sebagai file Excel error log.

### 4.3 Query Engine & Data Viewer
- **Dynamic Parameterized SQL Builder**:
  Menerjemahkan filter visual frontend menjadi SQL `WHERE` clause dengan parameter binding (`?`) untuk keamanan 100% dari SQL Injection.
- **Supported Filter Operators**:
  - String: `equals`, `not_equals`, `contains`, `not_contains`, `starts_with`, `ends_with`, `is_empty`, `is_not_empty`
  - Numeric: `=`, `!=`, `>`, `<`, `>=`, `<=`, `between`
  - Date: `equals`, `before`, `after`, `between`
- **Server-side Pagination**:
  `SELECT * FROM dataset_xxx WHERE ... ORDER BY col ASC/DESC LIMIT ? OFFSET ?`
  Query count terpisah atau dioptimalkan dengan `COUNT(*)` windowing.

### 4.4 Export Engine
- **Streaming XLSX Writer**: Menggunakan `excelize.StreamWriter` untuk menulis baris hasil query langsung ke file `.xlsx` di disk secara incremental.
- **Export Scopes**:
  - All Data (seluruh baris dataset)
  - Filtered Data (hanya baris yang memenuhi kriteria filter aktif)
  - Selected Rows (berdasarkan `_row_id` yang dicentang di frontend)
- **Custom Column Selection**: Pengguna dapat memilih subset kolom yang ingin di-export.

### 4.5 Keamanan & Autentikasi
- **Local SQLite Auth**: Login username dan password. Password di-hash menggunakan Argon2id/Bcrypt dengan salt acak.
- **Default Superadmin Initializer**: Pada first-run, jika user belum ada, dibuatkan admin default (`admin` / `admin123`) dengan rekomendasi ganti password.
- **Session State**: Token/session sederhana disimpan di memori runtime Go dan disinkronkan ke state Svelte.

### 4.6 Backup, Restore & Activity Logging
- **Database Backup**: Fasilitas export copy file `.db` dengan timestamp `natapadu_backup_YYYYMMDD_HHMMSS.db`.
- **Database Restore**: Validasi integritas file sebelum menggantikan database aktif.
- **Audit Logging**: Mencatat semua operasi penting (Login, Create Template, Import Job, Export File, Backup).

---

## 5. Rencana Rilis & Roadmap Build Multiplatform

- **Linux**: ELF Binary 64-bit (`natapadu`) & AppImage
- **Windows**: Executable x86_64 (`natapadu.exe`)
- **macOS**: Universal Binary / `.app` bundle
- **Automasi CI/CD**: GitHub Actions workflow untuk matrix build cross-compilation.
