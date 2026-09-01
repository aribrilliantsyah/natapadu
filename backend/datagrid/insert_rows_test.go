package datagrid

import (
	"path/filepath"
	"testing"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
	"natapadu-app/backend/template"
)

func setupInsertFixture(t *testing.T) (*DataGridService, *models.Template) {
	t.Helper()

	database, err := db.GetDatabase(filepath.Join(t.TempDir(), "natapadu_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	tplSvc := template.NewTemplateService(database)

	tpl, err := tplSvc.CreateTemplate(&models.Template{
		Name: "Peserta Massal", SheetName: "Sheet1", HeaderRow: 1, DataStartRow: 2,
		Columns: []models.TemplateColumn{
			{ExcelColumn: "A", FieldName: "nama", DisplayName: "Nama", DataType: "STRING"},
			{ExcelColumn: "B", FieldName: "umur", DisplayName: "Umur", DataType: "INTEGER"},
		},
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	return NewDataGridService(database, tplSvc), tpl
}

func countRows(t *testing.T, svc *DataGridService, tpl *models.Template) int64 {
	t.Helper()
	res, err := svc.QueryData(models.QueryRequest{TemplateID: tpl.ID, Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	return res.TotalRows
}

func TestInsertRowsSavesAll(t *testing.T) {
	svc, tpl := setupInsertFixture(t)

	n, err := svc.InsertRows(tpl.ID, []map[string]interface{}{
		{"nama": "Budi", "umur": int64(30)},
		{"nama": "Sinta", "umur": int64(25)},
		{"nama": "Rian", "umur": int64(41)},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if n != 3 {
		t.Errorf("tersimpan %d, mau 3", n)
	}
	if got := countRows(t, svc, tpl); got != 3 {
		t.Errorf("isi tabel %d baris, mau 3", got)
	}
}

// record_count di registry dataset harus ikut bertambah, bukan hanya tabelnya
func TestInsertRowsUpdatesRecordCount(t *testing.T) {
	svc, tpl := setupInsertFixture(t)

	if _, err := svc.InsertRows(tpl.ID, []map[string]interface{}{
		{"nama": "A"}, {"nama": "B"},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var count int64
	if err := svc.db.Conn().QueryRow(
		"SELECT record_count FROM datasets WHERE template_id = ?", tpl.ID,
	).Scan(&count); err != nil {
		t.Fatalf("baca record_count: %v", err)
	}
	if count != 2 {
		t.Errorf("record_count = %d, mau 2", count)
	}
}

// Satu baris gagal berarti tidak ada yang tersimpan — bukan separuh masuk,
// karena pengguna tidak akan tahu baris mana yang perlu diulang.
func TestInsertRowsIsAllOrNothing(t *testing.T) {
	svc, tpl := setupInsertFixture(t)

	// Kolom asing memicu kegagalan di tengah batch
	_, err := svc.InsertRows(tpl.ID, []map[string]interface{}{
		{"nama": "Budi", "umur": int64(30)},
		{"nama": make(chan int)}, // tipe yang tidak bisa ditulis ke SQLite
		{"nama": "Rian", "umur": int64(41)},
	})
	if err == nil {
		t.Fatal("harus gagal")
	}
	if got := countRows(t, svc, tpl); got != 0 {
		t.Errorf("%d baris tersimpan padahal batch gagal — transaksi tidak dibatalkan", got)
	}
}

func TestInsertRowsEmptyIsNoop(t *testing.T) {
	svc, tpl := setupInsertFixture(t)

	n, err := svc.InsertRows(tpl.ID, nil)
	if err != nil {
		t.Fatalf("insert kosong: %v", err)
	}
	if n != 0 {
		t.Errorf("tersimpan %d, mau 0", n)
	}
}

// Kolom yang tidak diisi harus tersimpan sebagai kosong, bukan menggeser nilai
// kolom lain — susunan kolom dikunci dari definisi template.
func TestInsertRowsHandlesMissingColumns(t *testing.T) {
	svc, tpl := setupInsertFixture(t)

	if _, err := svc.InsertRows(tpl.ID, []map[string]interface{}{
		{"nama": "Hanya Nama"},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	res, err := svc.QueryData(models.QueryRequest{TemplateID: tpl.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Data) != 1 {
		t.Fatalf("dapat %d baris", len(res.Data))
	}
	if res.Data[0]["nama"] != "Hanya Nama" {
		t.Errorf("nama = %v", res.Data[0]["nama"])
	}
	if v := res.Data[0]["umur"]; v != nil && v != "" {
		t.Errorf("umur = %v, mau kosong", v)
	}
}
