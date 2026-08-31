package template

import (
	"path/filepath"
	"testing"
	"time"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
)

// Pool SQLite dibatasi 1 koneksi (db.SetMaxOpenConns(1)). Query apa pun yang
// dijalankan sementara sebuah *sql.Rows masih terbuka akan menunggu koneksi yang
// belum dilepas — deadlock permanen, UI-nya loading selamanya. Test ini menjaga
// GetAllTemplates tetap menutup rows-nya sebelum mengambil data turunan.
func TestGetAllTemplatesDoesNotDeadlock(t *testing.T) {
	database, err := db.GetDatabase(filepath.Join(t.TempDir(), "natapadu_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	svc := NewTemplateService(database)

	if _, err := svc.CreateTemplate(&models.Template{
		Name:         "Data Peserta",
		SheetName:    "Sheet1",
		HeaderRow:    1,
		DataStartRow: 2,
		Columns: []models.TemplateColumn{
			{ExcelColumn: "A", FieldName: "nik", DisplayName: "NIK", DataType: "STRING", IsIndexed: true},
			{ExcelColumn: "B", FieldName: "nama", DisplayName: "Nama", DataType: "STRING"},
		},
	}); err != nil {
		t.Fatalf("create template: %v", err)
	}

	type result struct {
		list []models.Template
		err  error
	}
	done := make(chan result, 1)
	go func() {
		list, err := svc.GetAllTemplates()
		done <- result{list, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("GetAllTemplates: %v", r.err)
		}
		if len(r.list) != 1 {
			t.Fatalf("dapat %d workspace, mau 1", len(r.list))
		}
		if len(r.list[0].Columns) != 2 {
			t.Errorf("dapat %d kolom, mau 2", len(r.list[0].Columns))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetAllTemplates menggantung >5s — koneksi SQLite deadlock")
	}
}
