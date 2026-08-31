package exporter

import (
	"testing"

	"natapadu-app/backend/models"
)

// Template pengisian harus bisa di-import balik: header wajib mendarat di sheet,
// baris, dan huruf kolom yang sama persis dengan yang dibaca importer.
func TestBuildBlankTemplateFileHeaderPlacement(t *testing.T) {
	tpl := &models.Template{
		Name:         "Data Peserta",
		SheetName:    "Peserta",
		HeaderRow:    3, // header tidak di baris 1
		DataStartRow: 4,
		Columns: []models.TemplateColumn{
			{ExcelColumn: "A", DisplayName: "NIK", DataType: "STRING", Required: true},
			{ExcelColumn: "C", DisplayName: "Nama Lengkap", DataType: "STRING"}, // kolom loncat
			{ExcelColumn: "", DisplayName: "Tanggal Lahir", DataType: "DATE"},   // fallback ke urutan definisi
		},
	}

	f, err := buildBlankTemplateFile(tpl)
	if err != nil {
		t.Fatalf("buildBlankTemplateFile: %v", err)
	}

	want := map[string]string{
		"A3":  "NIK",
		"C3":  "Nama Lengkap",
		"C3_": "",
	}
	delete(want, "C3_")
	// kolom ketiga tanpa ExcelColumn -> jatuh ke huruf urutan ke-3 = "C"; pastikan tidak menimpa
	for cell, expect := range want {
		got, err := f.GetCellValue("Peserta", cell)
		if err != nil {
			t.Fatalf("GetCellValue(%s): %v", cell, err)
		}
		if got != expect {
			t.Errorf("sel %s = %q, mau %q", cell, got, expect)
		}
	}

	if v, _ := f.GetCellValue("Peserta", "A1"); v != "" {
		t.Errorf("baris 1 harus kosong saat headerRow=3, dapat %q", v)
	}
	if _, err := f.GetCellValue("Petunjuk Pengisian", "A1"); err != nil {
		t.Errorf("sheet petunjuk tidak ada: %v", err)
	}
	if idx, _ := f.GetSheetIndex("Sheet1"); idx != -1 {
		t.Errorf("Sheet1 bawaan harus dihapus saat sheetName kustom")
	}
}

// Template tanpa sheetName/headerRow harus tetap valid (fallback Sheet1 / baris 1).
func TestBuildBlankTemplateFileDefaults(t *testing.T) {
	tpl := &models.Template{
		Name:    "Tanpa Setelan",
		Columns: []models.TemplateColumn{{DisplayName: "Kolom Satu"}},
	}
	f, err := buildBlankTemplateFile(tpl)
	if err != nil {
		t.Fatalf("buildBlankTemplateFile: %v", err)
	}
	if got, _ := f.GetCellValue("Sheet1", "A1"); got != "Kolom Satu" {
		t.Errorf("Sheet1!A1 = %q, mau %q", got, "Kolom Satu")
	}
}
