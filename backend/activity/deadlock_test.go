package activity

import (
	"path/filepath"
	"testing"
	"time"

	"natapadu-app/backend/db"
)

// Sama seperti GetAllTemplates: dashboard menghitung isi tiap tabel dataset.
// Kalau COUNT dijalankan sementara rows daftar dataset masih terbuka, satu-satunya
// koneksi SQLite tidak pernah dilepas dan dashboard menggantung selamanya.
func TestGetDashboardSummaryDoesNotDeadlock(t *testing.T) {
	database, err := db.GetDatabase(filepath.Join(t.TempDir(), "natapadu_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Dua dataset terdaftar — deadlock butuh minimal satu baris di loop
	for _, name := range []string{"dataset_a", "dataset_b"} {
		if _, err := database.Conn().Exec("CREATE TABLE IF NOT EXISTS " + name + " (_row_id INTEGER PRIMARY KEY)"); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := database.Conn().Exec(
			"INSERT INTO templates (id, name, sheet_name, header_row, data_start_row) VALUES (?, ?, 'Sheet1', 1, 2)",
			name, name,
		); err != nil {
			t.Fatalf("register template %s: %v", name, err)
		}
		if _, err := database.Conn().Exec(
			"INSERT INTO datasets (id, template_id, table_name, record_count) VALUES (?, ?, ?, 0)",
			name, name, name,
		); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
		if _, err := database.Conn().Exec("INSERT INTO " + name + " (_row_id) VALUES (NULL)"); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	done := make(chan int64, 1)
	go func() {
		summary, err := NewActivityService(database).GetDashboardSummary()
		if err != nil || summary == nil {
			done <- -1
			return
		}
		done <- summary.TotalRecords
	}()

	select {
	case got := <-done:
		if got != 2 {
			t.Errorf("TotalRecords = %d, mau 2", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetDashboardSummary menggantung >5s — koneksi SQLite deadlock")
	}
}

// Agregat grafik: sumbu waktu harus penuh 14 hari (termasuk hari kosong),
// dan workspace diurut dari yang paling banyak barisnya.
func TestDashboardChartAggregates(t *testing.T) {
	database, err := db.GetDatabase(filepath.Join(t.TempDir(), "natapadu_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Dua workspace dengan jumlah baris berbeda
	sizes := map[string]int{"ws_kecil": 2, "ws_besar": 5}
	for name, n := range sizes {
		if _, err := database.Conn().Exec("CREATE TABLE IF NOT EXISTS " + name + " (_row_id INTEGER PRIMARY KEY)"); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := database.Conn().Exec(
			"INSERT INTO templates (id, name, sheet_name, header_row, data_start_row) VALUES (?, ?, 'Sheet1', 1, 2)", name, name,
		); err != nil {
			t.Fatalf("template %s: %v", name, err)
		}
		if _, err := database.Conn().Exec(
			"INSERT INTO datasets (id, template_id, table_name, record_count) VALUES (?, ?, ?, 0)", name, name, name,
		); err != nil {
			t.Fatalf("dataset %s: %v", name, err)
		}
		for i := 0; i < n; i++ {
			if _, err := database.Conn().Exec("INSERT INTO " + name + " (_row_id) VALUES (NULL)"); err != nil {
				t.Fatalf("seed %s: %v", name, err)
			}
		}
	}

	// Satu import hari ini
	if _, err := database.Conn().Exec(`
		INSERT INTO import_history (id, template_id, filename, total_rows, success_rows, failed_rows, imported_by, started_at, status)
		VALUES ('imp1', 'ws_besar', 'a.xlsx', 100, 90, 10, 'tester', datetime('now'), 'COMPLETED')
	`); err != nil {
		t.Fatalf("seed import: %v", err)
	}

	summary, err := NewActivityService(database).GetDashboardSummary()
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if len(summary.ImportTrend) != 14 {
		t.Errorf("ImportTrend = %d titik, mau 14 (hari kosong tetap diisi)", len(summary.ImportTrend))
	}
	last := summary.ImportTrend[len(summary.ImportTrend)-1]
	if last.Value != 90 || last.Secondary != 10 {
		t.Errorf("hari ini = %d sukses / %d gagal, mau 90/10", last.Value, last.Secondary)
	}
	if summary.SuccessRows != 90 || summary.FailedRows != 10 {
		t.Errorf("total = %d/%d, mau 90/10", summary.SuccessRows, summary.FailedRows)
	}

	// db.GetDatabase adalah singleton sync.Once, jadi paket ini berbagi satu database
	// antar test — periksa hanya workspace milik test ini, dan urutannya relatif.
	posBesar, posKecil := -1, -1
	for i, w := range summary.WorkspaceSizes {
		switch w.Label {
		case "ws_besar":
			posBesar = i
			if w.Value != 5 {
				t.Errorf("ws_besar = %d baris, mau 5", w.Value)
			}
		case "ws_kecil":
			posKecil = i
			if w.Value != 2 {
				t.Errorf("ws_kecil = %d baris, mau 2", w.Value)
			}
		}
	}
	if posBesar < 0 || posKecil < 0 {
		t.Fatalf("workspace tidak muncul di agregat: %+v", summary.WorkspaceSizes)
	}
	if posBesar > posKecil {
		t.Errorf("urutan salah — yang barisnya lebih banyak harus di depan: %+v", summary.WorkspaceSizes)
	}
}
