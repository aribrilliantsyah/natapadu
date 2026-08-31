package exporter

import (
	"archive/zip"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
	"natapadu-app/backend/template"

	"github.com/xuri/excelize/v2"
)

func setupExportFixture(t *testing.T) (*ExportService, *models.Template) {
	t.Helper()

	database, err := db.GetDatabase(filepath.Join(t.TempDir(), "natapadu_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	tplSvc := template.NewTemplateService(database)

	tpl, err := tplSvc.CreateTemplate(&models.Template{
		Name: "Data Peserta", SheetName: "Peserta", HeaderRow: 1, DataStartRow: 2,
		Columns: []models.TemplateColumn{
			{ExcelColumn: "A", FieldName: "perusahaan", DisplayName: "Perusahaan", DataType: "STRING", IsIndexed: true},
			{ExcelColumn: "B", FieldName: "nama", DisplayName: "Nama", DataType: "STRING"},
		},
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	tbl := template.GetTableNameForTemplate(tpl.ID)
	seed := [][2]string{
		{"PT Alpha", "Budi"},
		{"PT Alpha", "Sinta"},
		{"PT Beta", "Rian"},
		{"PT Beta, Tbk", `Nama "dikutip"`}, // koma & kutip: harus di-escape benar di CSV
	}
	for _, row := range seed {
		if _, err := database.Conn().Exec(
			"INSERT INTO "+tbl+" (_import_id, perusahaan, nama) VALUES ('TEST', ?, ?)", row[0], row[1],
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	return NewExportService(database, tplSvc), tpl
}

func TestExportXLSX(t *testing.T) {
	svc, tpl := setupExportFixture(t)
	dir := t.TempDir()

	res, err := svc.Export(models.ExportRequest{TemplateID: tpl.ID, Format: "XLSX", Scope: "ALL"}, dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if res.RowCount != 4 {
		t.Errorf("RowCount = %d, mau 4", res.RowCount)
	}
	if !strings.HasSuffix(res.FilePath, ".xlsx") {
		t.Errorf("ekstensi salah: %s", res.FilePath)
	}

	f, err := excelize.OpenFile(res.FilePath)
	if err != nil {
		t.Fatalf("buka hasil xlsx: %v", err)
	}
	defer f.Close()
	if got, _ := f.GetCellValue("Peserta", "A1"); got != "Perusahaan" {
		t.Errorf("A1 = %q, mau %q", got, "Perusahaan")
	}
	if got, _ := f.GetCellValue("Peserta", "B2"); got != "Budi" {
		t.Errorf("B2 = %q, mau %q", got, "Budi")
	}
}

func TestExportCSV(t *testing.T) {
	svc, tpl := setupExportFixture(t)
	dir := t.TempDir()

	res, err := svc.Export(models.ExportRequest{TemplateID: tpl.ID, Format: "CSV", Scope: "ALL"}, dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.HasSuffix(res.FilePath, ".csv") {
		t.Errorf("ekstensi salah: %s", res.FilePath)
	}

	raw, err := os.ReadFile(res.FilePath)
	if err != nil {
		t.Fatalf("baca csv: %v", err)
	}
	if !strings.HasPrefix(string(raw), "\xEF\xBB\xBF") {
		t.Error("CSV harus diawali BOM UTF-8 agar Excel Windows tidak merusak karakter")
	}

	recs, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\xEF\xBB\xBF"))).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(recs) != 5 { // 1 header + 4 data
		t.Fatalf("dapat %d baris, mau 5", len(recs))
	}
	if recs[0][0] != "Perusahaan" {
		t.Errorf("header = %q", recs[0][0])
	}
	// Baris dengan koma dan kutip harus utuh setelah round-trip
	if recs[4][0] != "PT Beta, Tbk" || recs[4][1] != `Nama "dikutip"` {
		t.Errorf("escaping rusak: %q / %q", recs[4][0], recs[4][1])
	}
}

func TestExportODS(t *testing.T) {
	svc, tpl := setupExportFixture(t)
	dir := t.TempDir()

	res, err := svc.Export(models.ExportRequest{TemplateID: tpl.ID, Format: "ODS", Scope: "ALL"}, dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.HasSuffix(res.FilePath, ".ods") {
		t.Errorf("ekstensi salah: %s", res.FilePath)
	}

	zr, err := zip.OpenReader(res.FilePath)
	if err != nil {
		t.Fatalf("ODS bukan zip yang sah: %v", err)
	}
	defer zr.Close()

	// Spesifikasi ODF: mimetype harus entri pertama dan tersimpan tanpa kompresi
	if len(zr.File) == 0 || zr.File[0].Name != "mimetype" {
		t.Fatal("entri pertama harus 'mimetype'")
	}
	if zr.File[0].Method != zip.Store {
		t.Error("mimetype harus disimpan tanpa kompresi")
	}

	contents := map[string]string{}
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatalf("buka %s: %v", zf.Name, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		contents[zf.Name] = string(b)
	}

	if contents["mimetype"] != "application/vnd.oasis.opendocument.spreadsheet" {
		t.Errorf("mimetype = %q", contents["mimetype"])
	}
	if _, ok := contents["META-INF/manifest.xml"]; !ok {
		t.Error("manifest.xml tidak ada")
	}

	body := contents["content.xml"]
	if !strings.Contains(body, `table:name="Peserta"`) {
		t.Error("nama sheet tidak terbawa")
	}
	if !strings.Contains(body, "<text:p>Perusahaan</text:p>") {
		t.Error("header tidak ditulis")
	}
	if strings.Count(body, "<table:table-row>") != 5 { // header + 4 data
		t.Errorf("dapat %d baris, mau 5", strings.Count(body, "<table:table-row>"))
	}
	// Karakter XML wajib di-escape, kalau tidak file-nya korup
	if !strings.Contains(body, "&#34;dikutip&#34;") && !strings.Contains(body, "&quot;dikutip&quot;") {
		t.Error("kutip harus di-escape di content.xml")
	}
}

func TestExportFilteredScope(t *testing.T) {
	svc, tpl := setupExportFixture(t)
	dir := t.TempDir()

	res, err := svc.Export(models.ExportRequest{
		TemplateID: tpl.ID, Format: "CSV", Scope: "FILTERED",
		Filters: []models.FilterCondition{{FieldName: "perusahaan", Operator: "equals", Value: "PT Alpha"}},
	}, dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if res.RowCount != 2 {
		t.Errorf("RowCount = %d, mau 2 (hanya PT Alpha)", res.RowCount)
	}
}

// Nama file berekstensi salah harus dikoreksi, bukan menghasilkan .xlsx berisi CSV
func TestExportFixesMismatchedExtension(t *testing.T) {
	svc, tpl := setupExportFixture(t)
	dir := t.TempDir()

	res, err := svc.Export(models.ExportRequest{
		TemplateID: tpl.ID, Format: "CSV", Scope: "ALL", OutputFilename: "hasil.xlsx",
	}, dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if filepath.Base(res.FilePath) != "hasil.csv" {
		t.Errorf("nama file = %q, mau hasil.csv", filepath.Base(res.FilePath))
	}
}
