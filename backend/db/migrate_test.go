package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// openRaw membuka database tanpa melewati singleton GetDatabase,
// supaya tiap test punya berkas sendiri.
func openRaw(t *testing.T, path string) *Database {
	t.Helper()
	conn, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	return &Database{db: conn, dbPath: path}
}

func TestMigrateFreshDatabase(t *testing.T) {
	d := openRaw(t, filepath.Join(t.TempDir(), "baru.db"))

	if err := d.migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	v, err := d.schemaVersion()
	if err != nil {
		t.Fatalf("schemaVersion: %v", err)
	}
	if want := len(d.migrations()); v != want {
		t.Errorf("versi skema = %d, mau %d", v, want)
	}

	for _, tbl := range []string{"users", "templates", "template_columns", "datasets", "app_settings"} {
		var n int
		if err := d.db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&n); err != nil || n != 1 {
			t.Errorf("tabel %s tidak terbentuk (n=%d, err=%v)", tbl, n, err)
		}
	}
}

// Menjalankan migrasi berulang kali tidak boleh mengubah apa pun —
// aplikasi memanggilnya setiap kali dibuka.
func TestMigrateIsIdempotent(t *testing.T) {
	d := openRaw(t, filepath.Join(t.TempDir(), "ulang.db"))

	if err := d.migrate(); err != nil {
		t.Fatalf("migrate pertama: %v", err)
	}
	if _, err := d.db.Exec(
		"INSERT INTO app_settings (key, value) VALUES ('theme', 'light')",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := d.migrate(); err != nil {
			t.Fatalf("migrate ulang #%d: %v", i, err)
		}
	}

	var val string
	if err := d.db.QueryRow("SELECT value FROM app_settings WHERE key='theme'").Scan(&val); err != nil {
		t.Fatalf("baca setelan: %v", err)
	}
	if val != "light" {
		t.Errorf("setelan = %q, mau light — data berubah saat migrasi diulang", val)
	}
}

// Inti dari "data lanjut dari versi lama": berkas yang dibuat versi aplikasi
// terdahulu (user_version = 0, tabel sudah ada, sudah berisi data) harus
// tetap utuh setelah aplikasi baru menjalankan migrasi.
func TestMigrateKeepsExistingUserData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lama.db")

	// Tiru database versi lama: skema awal, belum punya nomor versi
	old := openRaw(t, path)
	if _, err := old.db.Exec(baselineSchema); err != nil {
		t.Fatalf("skema lama: %v", err)
	}
	if _, err := old.db.Exec(`
		INSERT INTO templates (id, name, sheet_name, header_row, data_start_row)
		VALUES ('tpl1', 'Data Peserta', 'Sheet1', 1, 2)`); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if _, err := old.db.Exec(`
		INSERT INTO users (id, username, password_hash, display_name)
		VALUES ('u1', 'admin', 'hash', 'Administrator')`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if v, _ := old.schemaVersion(); v != 0 {
		t.Fatalf("database lama seharusnya versi 0, dapat %d", v)
	}

	// Aplikasi versi baru dibuka pada berkas yang sama
	fresh := openRaw(t, path)
	if err := fresh.migrate(); err != nil {
		t.Fatalf("migrate database lama: %v", err)
	}

	var name string
	if err := fresh.db.QueryRow("SELECT name FROM templates WHERE id='tpl1'").Scan(&name); err != nil {
		t.Fatalf("workspace lama hilang setelah migrasi: %v", err)
	}
	if name != "Data Peserta" {
		t.Errorf("nama workspace = %q, mau Data Peserta", name)
	}

	var uname string
	if err := fresh.db.QueryRow("SELECT username FROM users WHERE id='u1'").Scan(&uname); err != nil {
		t.Fatalf("user lama hilang setelah migrasi: %v", err)
	}

	if v, _ := fresh.schemaVersion(); v != len(fresh.migrations()) {
		t.Errorf("versi skema tidak dinaikkan: %d", v)
	}
}

// AddColumnIfMissing adalah jalur yang dipakai migrasi mendatang untuk
// menambah kolom pada tabel yang sudah berisi data pengguna.
func TestAddColumnIfMissing(t *testing.T) {
	d := openRaw(t, filepath.Join(t.TempDir(), "kolom.db"))
	if err := d.migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := d.db.Exec(
		"INSERT INTO templates (id, name, sheet_name, header_row, data_start_row) VALUES ('t1','Uji','Sheet1',1,2)",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := d.AddColumnIfMissing("templates", "catatan", "TEXT"); err != nil {
		t.Fatalf("tambah kolom: %v", err)
	}
	// Dipanggil dua kali harus tetap aman
	if err := d.AddColumnIfMissing("templates", "catatan", "TEXT"); err != nil {
		t.Fatalf("tambah kolom kedua kali: %v", err)
	}

	ada, err := d.columnExists("templates", "catatan")
	if err != nil || !ada {
		t.Fatalf("kolom tidak terpasang (ada=%v, err=%v)", ada, err)
	}

	var name string
	if err := d.db.QueryRow("SELECT name FROM templates WHERE id='t1'").Scan(&name); err != nil || name != "Uji" {
		t.Errorf("baris lama rusak setelah ALTER TABLE: %q %v", name, err)
	}
}

// Migrasi yang mengubah struktur harus menyisakan cadangan berkas.
func TestBackupBeforeMigrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cadangan.db")

	d := openRaw(t, path)
	if err := d.migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	target, err := d.backupBeforeMigrate(1)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if target == "" {
		t.Fatal("tidak ada cadangan yang dibuat")
	}
	if !strings.Contains(filepath.Base(target), "cadangan.db.v1-") {
		t.Errorf("nama cadangan tidak memuat versi asal: %s", target)
	}
	if fi, err := os.Stat(target); err != nil || fi.Size() == 0 {
		t.Errorf("berkas cadangan kosong atau tidak ada: %v", err)
	}
}
