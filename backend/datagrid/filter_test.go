package datagrid

import (
	"path/filepath"
	"testing"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
	"natapadu-app/backend/template"
)

// Dataset: dua baris kembar penuh (Budi/PT Alpha), satu nama sama tapi
// perusahaan beda (Budi/PT Beta) — membedakan duplikat 1 kolom vs 2 kolom.
func setupFilterFixture(t *testing.T) (*DataGridService, *models.Template) {
	t.Helper()

	database, err := db.GetDatabase(filepath.Join(t.TempDir(), "natapadu_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	tplSvc := template.NewTemplateService(database)

	tpl, err := tplSvc.CreateTemplate(&models.Template{
		Name: "Peserta", SheetName: "Sheet1", HeaderRow: 1, DataStartRow: 2,
		Columns: []models.TemplateColumn{
			{ExcelColumn: "A", FieldName: "nama", DisplayName: "Nama", DataType: "STRING", IsIndexed: true},
			{ExcelColumn: "B", FieldName: "perusahaan", DisplayName: "Perusahaan", DataType: "STRING", IsIndexed: true},
		},
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	tbl := template.GetTableNameForTemplate(tpl.ID)
	for _, r := range [][2]string{
		{"Budi", "PT Alpha"},
		{"Budi", "PT Alpha"},
		{"Budi", "PT Beta"},
		{"Sinta", "PT Beta"},
		{"Rian", "CV Gamma"},
	} {
		if _, err := database.Conn().Exec(
			"INSERT INTO "+tbl+" (_import_id, nama, perusahaan) VALUES ('TEST', ?, ?)", r[0], r[1],
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return NewDataGridService(database, tplSvc), tpl
}

func queryCount(t *testing.T, svc *DataGridService, tpl *models.Template, req models.QueryRequest) int64 {
	t.Helper()
	req.TemplateID = tpl.ID
	req.Page, req.PageSize = 1, 100
	res, err := svc.QueryData(req)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	return res.TotalRows
}

func TestFilterIsDuplicateSingleColumn(t *testing.T) {
	svc, tpl := setupFilterFixture(t)

	// nama "Budi" muncul 3x → ketiganya duplikat
	got := queryCount(t, svc, tpl, models.QueryRequest{
		Filters: []models.FilterCondition{{FieldName: "nama", Operator: "is_duplicate"}},
	})
	if got != 3 {
		t.Errorf("duplikat nama = %d baris, mau 3", got)
	}
}

func TestFilterIsDuplicateCompositeKey(t *testing.T) {
	svc, tpl := setupFilterFixture(t)

	// nama+perusahaan: hanya Budi/PT Alpha yang berulang → 2 baris
	got := queryCount(t, svc, tpl, models.QueryRequest{
		Filters: []models.FilterCondition{{FieldName: "nama", Operator: "is_duplicate", Value: "perusahaan"}},
	})
	if got != 2 {
		t.Errorf("duplikat nama+perusahaan = %d baris, mau 2", got)
	}
}

func TestFilterIsNotDuplicate(t *testing.T) {
	svc, tpl := setupFilterFixture(t)

	// Nama unik: Sinta, Rian
	got := queryCount(t, svc, tpl, models.QueryRequest{
		Filters: []models.FilterCondition{{FieldName: "nama", Operator: "is_not_duplicate"}},
	})
	if got != 2 {
		t.Errorf("nama unik = %d baris, mau 2", got)
	}
}

func TestFilterInList(t *testing.T) {
	svc, tpl := setupFilterFixture(t)

	got := queryCount(t, svc, tpl, models.QueryRequest{
		Filters: []models.FilterCondition{{FieldName: "perusahaan", Operator: "in_list", Value: "PT Beta, CV Gamma"}},
	})
	if got != 3 {
		t.Errorf("in_list = %d baris, mau 3", got)
	}

	got = queryCount(t, svc, tpl, models.QueryRequest{
		Filters: []models.FilterCondition{{FieldName: "perusahaan", Operator: "not_in_list", Value: "PT Beta\nCV Gamma"}},
	})
	if got != 2 {
		t.Errorf("not_in_list (dipisah baris baru) = %d baris, mau 2", got)
	}
}

func TestFilterLogicOR(t *testing.T) {
	svc, tpl := setupFilterFixture(t)

	filters := []models.FilterCondition{
		{FieldName: "nama", Operator: "equals", Value: "Sinta"},
		{FieldName: "perusahaan", Operator: "equals", Value: "CV Gamma"},
	}

	if got := queryCount(t, svc, tpl, models.QueryRequest{Filters: filters}); got != 0 {
		t.Errorf("AND = %d baris, mau 0 (tidak ada yang memenuhi keduanya)", got)
	}
	if got := queryCount(t, svc, tpl, models.QueryRequest{Filters: filters, FilterLogic: "OR"}); got != 2 {
		t.Errorf("OR = %d baris, mau 2", got)
	}
}

// Pencarian global harus tetap mempersempit hasil walau kondisi filter digabung OR
func TestFilterLogicORKeepsSearchNarrowing(t *testing.T) {
	svc, tpl := setupFilterFixture(t)

	got := queryCount(t, svc, tpl, models.QueryRequest{
		SearchTerm:  "Gamma",
		FilterLogic: "OR",
		Filters: []models.FilterCondition{
			{FieldName: "nama", Operator: "equals", Value: "Sinta"},
			{FieldName: "nama", Operator: "equals", Value: "Rian"},
		},
	})
	if got != 1 {
		t.Errorf("search + OR = %d baris, mau 1 (hanya Rian/CV Gamma)", got)
	}
}

func TestGetDuplicateGroups(t *testing.T) {
	svc, tpl := setupFilterFixture(t)

	groups, err := svc.GetDuplicateGroups(tpl.ID, []string{"nama"}, "", 0)
	if err != nil {
		t.Fatalf("duplicate groups: %v", err)
	}
	if len(groups) != 1 || groups[0].Values[0] != "Budi" || groups[0].Count != 3 {
		t.Fatalf("dapat %+v, mau satu grup Budi (3)", groups)
	}

	pair, err := svc.GetDuplicateGroups(tpl.ID, []string{"nama", "perusahaan"}, "", 0)
	if err != nil {
		t.Fatalf("duplicate groups gabungan: %v", err)
	}
	if len(pair) != 1 || pair[0].Count != 2 {
		t.Fatalf("dapat %+v, mau satu grup dengan 2 baris", pair)
	}

	if _, err := svc.GetDuplicateGroups(tpl.ID, []string{"kolom_palsu"}, "", 0); err == nil {
		t.Error("kolom tak dikenal harus ditolak")
	}
}
