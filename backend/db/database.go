package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db     *sql.DB
	dbPath string
	mu     sync.RWMutex
}

var (
	instance *Database
	once     sync.Once
)

// GetDatabase returns the singleton database instance
func GetDatabase(customPath ...string) (*Database, error) {
	var initErr error
	once.Do(func() {
		dbPath := ""
		if len(customPath) > 0 && customPath[0] != "" {
			dbPath = customPath[0]
		} else {
			userDir, err := os.UserConfigDir()
			if err != nil {
				userDir = "."
			}
			appDataDir := filepath.Join(userDir, "natapadu")
			_ = os.MkdirAll(appDataDir, 0755)
			dbPath = filepath.Join(appDataDir, "natapadu.db")
		}

		db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
		if err != nil {
			initErr = fmt.Errorf("failed to open sqlite database: %w", err)
			return
		}

		// Configure connection pool for optimum SQLite performance
		db.SetMaxOpenConns(1) // Single writer for SQLite to avoid lock contention
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(time.Hour)

		instance = &Database{
			db:     db,
			dbPath: dbPath,
		}

		if err := instance.migrate(); err != nil {
			initErr = fmt.Errorf("database migration failed: %w", err)
			return
		}
	})

	if initErr != nil {
		return nil, initErr
	}
	return instance, nil
}

// Conn returns the raw SQL DB connection
func (d *Database) Conn() *sql.DB {
	return d.db
}

// Path returns the physical database file path
func (d *Database) Path() string {
	return d.dbPath
}

// Close gracefully closes the database
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// FileSize returns the current size of the SQLite file in bytes
func (d *Database) FileSize() (int64, error) {
	info, err := os.Stat(d.dbPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// migrate initializes system tables
// baselineSchema adalah struktur awal aplikasi. Semua pernyataan memakai
// IF NOT EXISTS sehingga aman dijalankan pada database lama yang tabelnya
// sudah ada — dipakai sebagai migrasi #1.
const baselineSchema = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		display_name TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'USER',
		status TEXT NOT NULL DEFAULT 'ACTIVE',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		sheet_name TEXT NOT NULL DEFAULT 'Sheet1',
		header_row INTEGER NOT NULL DEFAULT 1,
		data_start_row INTEGER NOT NULL DEFAULT 2,
		version INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL DEFAULT 'ACTIVE',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS template_columns (
		id TEXT PRIMARY KEY,
		template_id TEXT NOT NULL,
		excel_column TEXT NOT NULL,
		field_name TEXT NOT NULL,
		display_name TEXT NOT NULL,
		data_type TEXT NOT NULL,
		format_pattern TEXT,
		required INTEGER NOT NULL DEFAULT 0,
		is_unique INTEGER NOT NULL DEFAULT 0,
		default_value TEXT,
		transform_rules TEXT,
		validation_rules TEXT,
		sort_order INTEGER NOT NULL DEFAULT 0,
		is_indexed INTEGER NOT NULL DEFAULT 1,
		FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS datasets (
		id TEXT PRIMARY KEY,
		template_id TEXT NOT NULL UNIQUE,
		table_name TEXT NOT NULL UNIQUE,
		record_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
	);

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
		status TEXT NOT NULL,
		error_message TEXT,
		FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
	);

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

	CREATE TABLE IF NOT EXISTS saved_filters (
		id TEXT PRIMARY KEY,
		template_id TEXT NOT NULL,
		name TEXT NOT NULL,
		filter_payload TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(template_id) REFERENCES templates(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS activity_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT,
		username TEXT,
		action TEXT NOT NULL,
		target TEXT,
		details TEXT,
		ip_address TEXT DEFAULT 'localhost',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS app_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_template_cols ON template_columns (template_id, sort_order);
	CREATE INDEX IF NOT EXISTS idx_import_hist_tpl ON import_history (template_id, started_at);
	CREATE INDEX IF NOT EXISTS idx_import_err_imp ON import_errors (import_id);
	CREATE INDEX IF NOT EXISTS idx_act_logs_created ON activity_logs (created_at DESC);
	`

// migrations berisi seluruh langkah perubahan skema secara berurutan.
// Menambah kolom baru di kemudian hari: tambahkan langkah BARU di akhir slice,
// jangan mengubah baselineSchema — CREATE TABLE IF NOT EXISTS tidak berpengaruh
// pada tabel yang sudah ada, sehingga kolomnya tidak akan pernah terpasang di
// database pengguna lama.
func (d *Database) migrations() []migration {
	return []migration{
		{name: "skema awal", sql: baselineSchema},
	}
}

func (d *Database) migrate() error {
	return d.runMigrations(d.migrations())
}
