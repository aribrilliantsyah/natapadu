package datagrid

import (
	"path/filepath"
	"testing"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
	"natapadu-app/backend/template"
)

func TestGetDistinctValues(t *testing.T) {
	database, err := db.GetDatabase(filepath.Join(t.TempDir(), "natapadu_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	tplSvc := template.NewTemplateService(database)

	tpl, err := tplSvc.CreateTemplate(&models.Template{
		Name: "Data Peserta", SheetName: "Sheet1", HeaderRow: 1, DataStartRow: 2,
		Columns: []models.TemplateColumn{
			{ExcelColumn: "A", FieldName: "perusahaan", DisplayName: "Perusahaan", DataType: "STRING", IsIndexed: true},
		},
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	tbl := template.GetTableNameForTemplate(tpl.ID)
	for _, v := range []string{"PT Alpha", "PT Alpha", "PT Alpha", "PT Beta", "CV Gamma"} {
		if _, err := database.Conn().Exec("INSERT INTO "+tbl+" (_import_id, perusahaan) VALUES ('TEST', ?)", v); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	svc := NewDataGridService(database, tplSvc)

	got, err := svc.GetDistinctValues(tpl.ID, "perusahaan", "", 0)
	if err != nil {
		t.Fatalf("distinct: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("dapat %d nilai unik, mau 3", len(got))
	}
	// Diurut jumlah terbanyak dulu
	if got[0].Value != "PT Alpha" || got[0].Count != 3 {
		t.Errorf("teratas = %q (%d), mau PT Alpha (3)", got[0].Value, got[0].Count)
	}

	only, err := svc.GetDistinctValues(tpl.ID, "perusahaan", "Beta", 0)
	if err != nil {
		t.Fatalf("distinct dengan pencarian: %v", err)
	}
	if len(only) != 1 || only[0].Value != "PT Beta" {
		t.Errorf("pencarian 'Beta' dapat %+v", only)
	}

	// Kolom di luar template harus ditolak, bukan diteruskan ke SQL
	if _, err := svc.GetDistinctValues(tpl.ID, "perusahaan; DROP TABLE users", "", 0); err == nil {
		t.Error("kolom tak dikenal harus ditolak")
	}
}
