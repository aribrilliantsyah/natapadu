package db

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// migration adalah satu langkah perubahan skema yang dijalankan tepat sekali.
// Urutan slice menentukan urutan eksekusi; JANGAN pernah menyisipkan langkah
// di tengah atau mengubah langkah yang sudah dirilis — tambahkan di akhir saja,
// karena database pengguna sudah menyimpan sampai langkah ke berapa ia berjalan.
type migration struct {
	name string
	sql  string
	fn   func(d *Database) error // dipakai bila perubahannya butuh logika, bukan SQL statis
}

// schemaVersion dilacak lewat PRAGMA user_version bawaan SQLite —
// tidak perlu tabel tambahan, dan ikut serta di dalam berkas .db itu sendiri.
func (d *Database) schemaVersion() (int, error) {
	var v int
	err := d.db.QueryRow("PRAGMA user_version").Scan(&v)
	return v, err
}

func (d *Database) setSchemaVersion(v int) error {
	// PRAGMA tidak menerima parameter berparameter, dan v selalu berasal dari
	// konstanta di kode ini — tidak pernah dari input pengguna.
	_, err := d.db.Exec(fmt.Sprintf("PRAGMA user_version = %d", v))
	return err
}

// runMigrations menjalankan setiap langkah yang belum pernah dijalankan pada
// database ini, sehingga berkas dari versi aplikasi lama tetap terpakai apa
// adanya — datanya tidak pernah dibuat ulang.
func (d *Database) runMigrations(steps []migration) error {
	current, err := d.schemaVersion()
	if err != nil {
		return fmt.Errorf("gagal membaca versi skema: %w", err)
	}
	if current >= len(steps) {
		return nil // sudah mutakhir
	}

	// Cadangkan sebelum mengubah struktur. Hanya berjalan saat benar-benar ada
	// migrasi baru, jadi biayanya ditanggung sekali per pembaruan aplikasi.
	if current > 0 {
		if path, err := d.backupBeforeMigrate(current); err != nil {
			return fmt.Errorf("gagal mencadangkan sebelum migrasi: %w", err)
		} else if path != "" {
			fmt.Printf("[migrasi] cadangan dibuat: %s\n", path)
		}
	}

	for i := current; i < len(steps); i++ {
		step := steps[i]
		version := i + 1

		if err := d.applyMigration(step); err != nil {
			return fmt.Errorf("migrasi #%d (%s) gagal: %w", version, step.name, err)
		}
		if err := d.setSchemaVersion(version); err != nil {
			return fmt.Errorf("gagal menyimpan versi skema %d: %w", version, err)
		}
		fmt.Printf("[migrasi] #%d %s selesai\n", version, step.name)
	}
	return nil
}

// applyMigration menjalankan satu langkah dalam transaksi supaya tidak ada
// keadaan setengah jadi bila terputus di tengah jalan.
func (d *Database) applyMigration(step migration) error {
	if step.fn != nil {
		return step.fn(d)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(step.sql); err != nil {
		return err
	}
	return tx.Commit()
}

// backupBeforeMigrate menyalin berkas database aktif berdampingan dengan aslinya.
// Cadangan lama tidak pernah ditimpa karena namanya memuat stempel waktu.
func (d *Database) backupBeforeMigrate(fromVersion int) (string, error) {
	src, err := os.Open(d.dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // database baru, tidak ada yang perlu dicadangkan
		}
		return "", err
	}
	defer src.Close()

	dir := filepath.Dir(d.dbPath)
	base := filepath.Base(d.dbPath)
	target := filepath.Join(dir, fmt.Sprintf("%s.v%d-%s.bak", base, fromVersion, time.Now().Format("20060102_150405")))

	dst, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return target, dst.Sync()
}

// columnExists dipakai langkah migrasi yang menambah kolom, karena SQLite
// tidak punya ALTER TABLE ... ADD COLUMN IF NOT EXISTS.
func (d *Database) columnExists(table, column string) (bool, error) {
	rows, err := d.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			continue
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// AddColumnIfMissing dipakai di dalam langkah migrasi untuk menambah kolom
// pada tabel yang sudah berisi data pengguna, tanpa menyentuh isinya.
func (d *Database) AddColumnIfMissing(table, column, definition string) error {
	exists, err := d.columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = d.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}
