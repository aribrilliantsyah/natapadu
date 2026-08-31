package exporter

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"natapadu-app/backend/datagrid"
	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
	"natapadu-app/backend/template"

	"github.com/xuri/excelize/v2"
)

type ExportService struct {
	db          *db.Database
	templateSvc *template.TemplateService
}

func NewExportService(database *db.Database, tplSvc *template.TemplateService) *ExportService {
	return &ExportService{
		db:          database,
		templateSvc: tplSvc,
	}
}

// Export menjalankan query dataset lalu menuliskannya ke berkas sesuai format yang diminta.
// Mengembalikan struct (bukan tiga nilai) karena binding Wails hanya mendukung (value, error).
func (s *ExportService) Export(req models.ExportRequest, saveDirectory string) (*models.ExportResult, error) {
	if req.TemplateID == "" {
		return nil, errors.New("template ID is required")
	}

	tpl, err := s.templateSvc.GetTemplateByID(req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template tidak ditemukan: %w", err)
	}

	exportCols, err := resolveExportColumns(tpl, req.Columns)
	if err != nil {
		return nil, err
	}

	querySQL, args, err := buildExportQuery(tpl, exportCols, req)
	if err != nil {
		return nil, err
	}

	format := normalizeFormat(req.Format)
	outputPath, err := resolveOutputPath(tpl, req.OutputFilename, saveDirectory, format)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Conn().Query(querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query data export: %w", err)
	}
	defer rows.Close()

	headers := make([]string, len(exportCols))
	for i, c := range exportCols {
		headers[i] = c.DisplayName
	}

	var rowCount int64
	switch format {
	case "CSV":
		rowCount, err = writeCSV(outputPath, headers, rows)
	case "ODS":
		rowCount, err = writeODS(outputPath, tpl.SheetName, headers, rows)
	default:
		rowCount, err = writeXLSX(outputPath, tpl.SheetName, headers, rows)
	}
	if err != nil {
		return nil, err
	}

	return &models.ExportResult{FilePath: outputPath, RowCount: rowCount, Format: format}, nil
}

// normalizeFormat memetakan input pengguna ke salah satu format yang didukung
func normalizeFormat(f string) string {
	switch strings.ToUpper(strings.TrimSpace(f)) {
	case "CSV":
		return "CSV"
	case "ODS", "OPENDOCUMENT":
		return "ODS"
	default:
		return "XLSX"
	}
}

func formatExtension(format string) string {
	switch format {
	case "CSV":
		return ".csv"
	case "ODS":
		return ".ods"
	default:
		return ".xlsx"
	}
}

// resolveExportColumns memilih kolom yang diekspor, mempertahankan urutan permintaan
func resolveExportColumns(tpl *models.Template, wanted []string) ([]models.TemplateColumn, error) {
	if len(wanted) == 0 {
		if len(tpl.Columns) == 0 {
			return nil, errors.New("tidak ada kolom yang dipilih untuk di-export")
		}
		return tpl.Columns, nil
	}

	colMap := make(map[string]models.TemplateColumn, len(tpl.Columns))
	for _, c := range tpl.Columns {
		colMap[strings.ToLower(c.FieldName)] = c
	}

	var out []models.TemplateColumn
	for _, name := range wanted {
		if c, ok := colMap[strings.ToLower(name)]; ok {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("tidak ada kolom yang dipilih untuk di-export")
	}
	return out, nil
}

// buildExportQuery menyusun SELECT beserta WHERE/ORDER BY sesuai cakupan export
func buildExportQuery(tpl *models.Template, exportCols []models.TemplateColumn, req models.ExportRequest) (string, []interface{}, error) {
	tableName := template.GetTableNameForTemplate(tpl.ID)

	var whereSQL string
	var args []interface{}

	switch strings.ToUpper(req.Scope) {
	case "SELECTED":
		if len(req.SelectedRowIDs) == 0 {
			return "", nil, errors.New("tidak ada baris yang dipilih untuk di-export")
		}
		placeholders := make([]string, len(req.SelectedRowIDs))
		for i, id := range req.SelectedRowIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		whereSQL = fmt.Sprintf(" WHERE _row_id IN (%s)", strings.Join(placeholders, ", "))

	case "FILTERED":
		w, a, err := datagrid.BuildWhereClause(tableName, tpl.Columns, req.Filters, req.SearchTerm, req.FilterLogic)
		if err != nil {
			return "", nil, err
		}
		whereSQL = w
		args = a
	}

	sortSQL := " ORDER BY _row_id ASC"
	if req.SortBy != "" {
		dir := "ASC"
		if strings.EqualFold(req.SortOrder, "DESC") {
			dir = "DESC"
		}
		for _, c := range tpl.Columns {
			if strings.EqualFold(c.FieldName, req.SortBy) {
				sortSQL = fmt.Sprintf(" ORDER BY [%s] %s", c.FieldName, dir)
				break
			}
		}
	}

	selectCols := make([]string, len(exportCols))
	for i, c := range exportCols {
		selectCols[i] = fmt.Sprintf("[%s]", c.FieldName)
	}

	return fmt.Sprintf("SELECT %s FROM %s%s%s", strings.Join(selectCols, ", "), tableName, whereSQL, sortSQL), args, nil
}

// resolveOutputPath menentukan lokasi berkas akhir dan memastikan ekstensinya sesuai format
func resolveOutputPath(tpl *models.Template, filename, saveDirectory, format string) (string, error) {
	if saveDirectory == "" {
		userHome, _ := os.UserHomeDir()
		saveDirectory = filepath.Join(userHome, "Downloads")
	}
	if err := os.MkdirAll(saveDirectory, 0755); err != nil {
		return "", fmt.Errorf("gagal menyiapkan folder output: %w", err)
	}

	ext := formatExtension(format)
	if filename == "" {
		filename = fmt.Sprintf("Export_%s_%s%s", template.SanitizeIdentifier(tpl.Name), time.Now().Format("20060102_150405"), ext)
	}
	// Ganti ekstensi lain agar tidak ada file .xlsx yang isinya CSV
	if cur := strings.ToLower(filepath.Ext(filename)); cur != ext {
		if cur == ".xlsx" || cur == ".csv" || cur == ".ods" {
			filename = strings.TrimSuffix(filename, filepath.Ext(filename))
		}
		filename += ext
	}

	return filepath.Join(saveDirectory, filename), nil
}

// scanRowStrings membaca satu baris hasil query menjadi slice string siap tulis
func scanRowStrings(rows *sql.Rows, n int) ([]string, error) {
	vals := make([]interface{}, n)
	ptrs := make([]interface{}, n)
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}

	out := make([]string, n)
	for i, v := range vals {
		switch t := v.(type) {
		case nil:
			out[i] = ""
		case []byte:
			out[i] = string(t)
		case string:
			out[i] = t
		case float64:
			out[i] = strconv.FormatFloat(t, 'f', -1, 64)
		case int64:
			out[i] = strconv.FormatInt(t, 10)
		default:
			out[i] = fmt.Sprint(t)
		}
	}
	return out, nil
}

// writeXLSX menulis lewat StreamWriter excelize agar konsumsi memori tetap rendah
func writeXLSX(outputPath, sheetName string, headers []string, rows *sql.Rows) (int64, error) {
	if sheetName == "" {
		sheetName = "Data"
	}

	f := excelize.NewFile()
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return 0, fmt.Errorf("gagal membuat sheet: %w", err)
	}
	f.SetActiveSheet(index)
	if sheetName != "Sheet1" {
		_ = f.DeleteSheet("Sheet1")
	}

	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return 0, fmt.Errorf("gagal inisialisasi stream writer Excel: %w", err)
	}

	headerRow := make([]interface{}, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	if err := sw.SetRow("A1", headerRow); err != nil {
		return 0, fmt.Errorf("gagal menulis header Excel: %w", err)
	}

	var count int64
	excelRowIdx := 2
	for rows.Next() {
		vals, err := scanRowStrings(rows, len(headers))
		if err != nil {
			continue
		}
		cells := make([]interface{}, len(vals))
		for i, v := range vals {
			cells[i] = v
		}
		cellName, _ := excelize.CoordinatesToCellName(1, excelRowIdx)
		if err := sw.SetRow(cellName, cells); err != nil {
			return 0, fmt.Errorf("gagal menulis baris %d ke Excel: %w", excelRowIdx, err)
		}
		excelRowIdx++
		count++
	}

	if err := sw.Flush(); err != nil {
		return 0, fmt.Errorf("gagal flush stream Excel: %w", err)
	}
	if err := f.SaveAs(outputPath); err != nil {
		return 0, fmt.Errorf("gagal menyimpan file Excel ke disk: %w", err)
	}
	return count, nil
}

// writeCSV menulis CSV UTF-8 dengan BOM supaya Excel di Windows tidak merusak karakter non-ASCII
func writeCSV(outputPath string, headers []string, rows *sql.Rows) (int64, error) {
	f, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("gagal membuat file CSV: %w", err)
	}
	defer f.Close()

	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return 0, err
	}

	w := csv.NewWriter(f)
	if err := w.Write(headers); err != nil {
		return 0, fmt.Errorf("gagal menulis header CSV: %w", err)
	}

	var count int64
	for rows.Next() {
		vals, err := scanRowStrings(rows, len(headers))
		if err != nil {
			continue
		}
		if err := w.Write(vals); err != nil {
			return 0, fmt.Errorf("gagal menulis baris CSV: %w", err)
		}
		count++
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return 0, fmt.Errorf("gagal flush CSV: %w", err)
	}
	return count, nil
}

// ExportBlankTemplate menulis file XLSX kosong berisi baris header sesuai layout template
// (sheet, posisi kolom Excel, dan nomor baris header), plus satu sheet "Petunjuk Pengisian".
// Tujuannya supaya workspace yang sudah dibuat bisa diisi ulang tanpa menebak susunan kolom.
func (s *ExportService) ExportBlankTemplate(templateID, saveDirectory string) (string, error) {
	if templateID == "" {
		return "", errors.New("template ID is required")
	}

	tpl, err := s.templateSvc.GetTemplateByID(templateID)
	if err != nil {
		return "", fmt.Errorf("template tidak ditemukan: %w", err)
	}
	if len(tpl.Columns) == 0 {
		return "", errors.New("workspace ini belum punya definisi kolom")
	}

	f, err := buildBlankTemplateFile(tpl)
	if err != nil {
		return "", err
	}

	if saveDirectory == "" {
		userHome, _ := os.UserHomeDir()
		saveDirectory = filepath.Join(userHome, "Downloads")
	}
	_ = os.MkdirAll(saveDirectory, 0755)

	filename := fmt.Sprintf("Template_%s_%s.xlsx", template.SanitizeIdentifier(tpl.Name), time.Now().Format("20060102_150405"))
	fullOutputPath := filepath.Join(saveDirectory, filename)
	if err := f.SaveAs(fullOutputPath); err != nil {
		return "", fmt.Errorf("gagal menyimpan template Excel ke disk: %w", err)
	}

	return fullOutputPath, nil
}

// buildBlankTemplateFile menyusun workbook kosong sesuai layout template.
// Dipisah dari ExportBlankTemplate supaya bisa diuji tanpa database.
func buildBlankTemplateFile(tpl *models.Template) (*excelize.File, error) {
	sheetName := tpl.SheetName
	if sheetName == "" {
		sheetName = "Sheet1"
	}
	headerRow := tpl.HeaderRow
	if headerRow < 1 {
		headerRow = 1
	}

	f := excelize.NewFile()
	if sheetName != "Sheet1" {
		idx, err := f.NewSheet(sheetName)
		if err != nil {
			return nil, fmt.Errorf("gagal membuat sheet '%s': %w", sheetName, err)
		}
		f.SetActiveSheet(idx)
		_ = f.DeleteSheet("Sheet1")
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2563EB"}},
	})

	assigned, err := assignHeaderColumns(tpl.Columns)
	if err != nil {
		return nil, err
	}

	for i, c := range tpl.Columns {
		colName := assigned[i]
		cell, err := excelize.JoinCellName(colName, headerRow)
		if err != nil {
			continue
		}
		_ = f.SetCellStr(sheetName, cell, c.DisplayName)
		if headerStyle != 0 {
			_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
		_ = f.SetColWidth(sheetName, colName, colName, 22)
	}

	// Sheet petunjuk: tipe data & aturan validasi, biar baris tidak gagal saat import
	const guideSheet = "Petunjuk Pengisian"
	if _, err := f.NewSheet(guideSheet); err == nil {
		guide := [][]interface{}{
			{"Workspace", tpl.Name},
			{"Sheet data", sheetName},
			{"Baris header", headerRow},
			{"Baris data mulai", tpl.DataStartRow},
			{},
			{"Kolom Excel", "Nama Kolom", "Tipe Data", "Format", "Wajib", "Unik", "Transformasi"},
		}
		for i, c := range tpl.Columns {
			guide = append(guide, []interface{}{
				assigned[i], c.DisplayName, c.DataType, c.FormatPattern,
				boolLabel(c.Required), boolLabel(c.IsUnique), c.TransformRules,
			})
		}
		for i, row := range guide {
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			_ = f.SetSheetRow(guideSheet, cell, &row)
		}
		_ = f.SetColWidth(guideSheet, "A", "G", 20)
	}

	return f, nil
}

// assignHeaderColumns memetakan tiap kolom template ke huruf kolom Excel.
// Kolom dengan ExcelColumn eksplisit dikunci dulu, baru sisanya mengisi slot yang masih
// kosong — kalau tidak, kolom tanpa huruf akan menimpa header tetangganya.
func assignHeaderColumns(cols []models.TemplateColumn) ([]string, error) {
	assigned := make([]string, len(cols))
	used := make(map[string]bool, len(cols))

	for i, c := range cols {
		name := strings.ToUpper(strings.TrimSpace(c.ExcelColumn))
		if name == "" {
			continue
		}
		assigned[i] = name
		used[name] = true
	}

	next := 1
	for i := range cols {
		if assigned[i] != "" {
			continue
		}
		for {
			name, err := excelize.ColumnNumberToName(next)
			if err != nil {
				return nil, fmt.Errorf("jumlah kolom melebihi batas Excel: %w", err)
			}
			next++
			if !used[name] {
				assigned[i] = name
				used[name] = true
				break
			}
		}
	}
	return assigned, nil
}

func boolLabel(v bool) string {
	if v {
		return "Ya"
	}
	return "Tidak"
}
