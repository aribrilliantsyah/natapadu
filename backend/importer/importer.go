package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
	"natapadu-app/backend/template"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type ImportService struct {
	db          *db.Database
	templateSvc *template.TemplateService
	activeJobs  sync.Map // importId -> context.CancelFunc
}

func NewImportService(database *db.Database, tplSvc *template.TemplateService) *ImportService {
	return &ImportService{
		db:          database,
		templateSvc: tplSvc,
	}
}

// PreviewExcelFile reads sheet names, header rows, and sample data from an Excel file
func (s *ImportService) PreviewExcelFile(filePath, sheetName string, headerRow, sampleLimit int) (*models.ExcelSheetPreview, error) {
	if filePath == "" {
		return nil, errors.New("file path tidak boleh kosong")
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka file Excel: %w", err)
	}
	defer f.Close()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, errors.New("file Excel tidak memiliki sheet")
	}

	activeSheet := sheetName
	if activeSheet == "" {
		activeSheet = sheetList[0]
	}

	if headerRow <= 0 {
		headerRow = 1
	}
	if sampleLimit <= 0 {
		sampleLimit = 15
	}

	rows, err := f.Rows(activeSheet)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca baris sheet '%s': %w", activeSheet, err)
	}
	defer rows.Close()

	var headers []string
	var sampleRows [][]string
	currentRow := 0
	totalColumns := 0

	for rows.Next() {
		currentRow++
		rowCells, err := rows.Columns()
		if err != nil {
			continue
		}

		if len(rowCells) > totalColumns {
			totalColumns = len(rowCells)
		}

		if currentRow == headerRow {
			headers = make([]string, len(rowCells))
			copy(headers, rowCells)
		} else if currentRow > headerRow && len(sampleRows) < sampleLimit {
			sampleRows = append(sampleRows, rowCells)
		}

		// Don't read indefinitely for preview
		if currentRow > headerRow+sampleLimit+50 {
			break
		}
	}

	return &models.ExcelSheetPreview{
		Sheets:       sheetList,
		ActiveSheet:  activeSheet,
		TotalRows:    currentRow,
		TotalColumns: totalColumns,
		HeaderRow:    headerRow,
		DataStartRow: headerRow + 1,
		Headers:      headers,
		SampleRows:   sampleRows,
	}, nil
}

// ConvertExcelColumnNameToIndex converts column name like "A" -> 0, "B" -> 1, "AA" -> 26
func ConvertExcelColumnNameToIndex(colName string) int {
	colName = strings.ToUpper(strings.TrimSpace(colName))
	idx := 0
	for i := 0; i < len(colName); i++ {
		idx = idx*26 + int(colName[i]-'A'+1)
	}
	return idx - 1
}

// ConvertIndexToExcelColumnName converts 0 -> "A", 1 -> "B", 26 -> "AA"
func ConvertIndexToExcelColumnName(index int) string {
	col := ""
	for index >= 0 {
		col = string(rune('A'+(index%26))) + col
		index = index/26 - 1
	}
	return col
}

// TransformAndValidate applies data transformations and validation rules
func (s *ImportService) TransformAndValidate(rawVal string, col models.TemplateColumn) (interface{}, error) {
	val := rawVal

	// Parse transform rules
	var transforms []string
	if col.TransformRules != "" {
		_ = json.Unmarshal([]byte(col.TransformRules), &transforms)
	}

	for _, rule := range transforms {
		switch strings.ToUpper(rule) {
		case "TRIM":
			val = strings.TrimSpace(val)
		case "UPPERCASE":
			val = strings.ToUpper(val)
		case "LOWERCASE":
			val = strings.ToLower(val)
		case "CAPITALIZE":
			val = strings.Title(strings.ToLower(val))
		case "REMOVE_SPACE":
			val = strings.ReplaceAll(val, " ", "")
		case "NUMERIC_ONLY":
			re := regexp.MustCompile(`[^0-9]`)
			val = re.ReplaceAllString(val, "")
		}
	}

	// Default value if empty
	if strings.TrimSpace(val) == "" && col.DefaultValue != "" {
		val = col.DefaultValue
	}

	// Required Check
	if col.Required && strings.TrimSpace(val) == "" {
		return nil, fmt.Errorf("kolom '%s' (%s) wajib diisi", col.DisplayName, col.FieldName)
	}

	// If empty and not required, return nil (NULL in SQLite)
	if strings.TrimSpace(val) == "" {
		return nil, nil
	}

	// Parse validation rules
	if col.ValidationRules != "" {
		var rules map[string]interface{}
		if err := json.Unmarshal([]byte(col.ValidationRules), &rules); err == nil {
			if minL, ok := rules["minLength"].(float64); ok && float64(len(val)) < minL {
				return nil, fmt.Errorf("panjang minimal kolom '%s' adalah %d karakter", col.DisplayName, int(minL))
			}
			if maxL, ok := rules["maxLength"].(float64); ok && float64(len(val)) > maxL {
				return nil, fmt.Errorf("panjang maksimal kolom '%s' adalah %d karakter", col.DisplayName, int(maxL))
			}
			if exactL, ok := rules["exactLength"].(float64); ok && float64(len(val)) != exactL {
				return nil, fmt.Errorf("panjang kolom '%s' harus tepat %d karakter", col.DisplayName, int(exactL))
			}
			if regexPattern, ok := rules["regex"].(string); ok && regexPattern != "" {
				if matched, _ := regexp.MatchString(regexPattern, val); !matched {
					return nil, fmt.Errorf("format kolom '%s' tidak valid sesuai pola", col.DisplayName)
				}
			}
		}
	}

	// Type Casting
	switch strings.ToUpper(col.DataType) {
	case "INTEGER":
		clean := strings.ReplaceAll(strings.ReplaceAll(val, ",", ""), ".", "")
		num, err := strconv.ParseInt(clean, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("nilai '%s' bukan integer yang valid", val)
		}
		return num, nil

	case "DECIMAL", "PERCENTAGE":
		clean := strings.ReplaceAll(val, ",", ".")
		clean = strings.ReplaceAll(clean, "%", "")
		num, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return nil, fmt.Errorf("nilai '%s' bukan angka desimal yang valid", val)
		}
		return num, nil

	case "CURRENCY":
		clean := strings.ToUpper(val)
		clean = strings.ReplaceAll(clean, "RP", "")
		clean = strings.ReplaceAll(clean, "IDR", "")
		clean = strings.ReplaceAll(clean, " ", "")
		clean = strings.ReplaceAll(clean, ".", "")
		clean = strings.ReplaceAll(clean, ",", ".")
		num, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return nil, fmt.Errorf("nilai uang '%s' tidak valid", val)
		}
		return num, nil

	case "BOOLEAN":
		lower := strings.ToLower(strings.TrimSpace(val))
		if lower == "true" || lower == "1" || lower == "ya" || lower == "y" || lower == "yes" {
			return 1, nil
		} else if lower == "false" || lower == "0" || lower == "tidak" || lower == "t" || lower == "no" {
			return 0, nil
		}
		return nil, fmt.Errorf("nilai '%s' bukan boolean valid", val)

	case "DATE", "DATETIME":
		parsedDate := parseFlexibleDate(val, col.FormatPattern)
		if parsedDate == "" {
			return nil, fmt.Errorf("format tanggal '%s' tidak dikenali", val)
		}
		return parsedDate, nil

	default: // STRING
		return val, nil
	}
}

func parseFlexibleDate(val, pattern string) string {
	v := strings.TrimSpace(val)
	if v == "" {
		return ""
	}

	// Try standard ISO
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t.Format("2006-01-02")
	}
	if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	if t, err := time.Parse("02/01/2006", v); err == nil {
		return t.Format("2006-01-02")
	}
	if t, err := time.Parse("02-01-2006", v); err == nil {
		return t.Format("2006-01-02")
	}
	if t, err := time.Parse("2/1/2006", v); err == nil {
		return t.Format("2006-01-02")
	}
	if t, err := time.Parse("02/01/2006 15:04:05", v); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}

	// Excel float serial date check (e.g. 45000 -> date)
	if floatVal, err := strconv.ParseFloat(v, 64); err == nil && floatVal > 20000 && floatVal < 90000 {
		excelEpoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		days := int(floatVal)
		t := excelEpoch.AddDate(0, 0, days)
		return t.Format("2006-01-02")
	}

	return v
}

// ExecuteImport performs high-speed streaming batch ingestion into SQLite
func (s *ImportService) ExecuteImport(
	ctx context.Context,
	templateID string,
	filePath string,
	importedBy string,
	progressCallback func(models.ImportProgressEvent),
) (*models.ImportHistory, error) {

	tpl, err := s.templateSvc.GetTemplateByID(templateID)
	if err != nil {
		return nil, fmt.Errorf("template tidak ditemukan: %w", err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file tidak dapat diakses: %w", err)
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka excel: %w", err)
	}
	defer f.Close()

	sheetName := tpl.SheetName
	sheetList := f.GetSheetList()
	sheetFound := false
	for _, sh := range sheetList {
		if strings.EqualFold(sh, sheetName) {
			sheetName = sh
			sheetFound = true
			break
		}
	}
	if !sheetFound && len(sheetList) > 0 {
		sheetName = sheetList[0]
	}

	rows, err := f.Rows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka stream baris sheet: %w", err)
	}
	defer rows.Close()

	importID := uuid.New().String()
	history := &models.ImportHistory{
		ID:            importID,
		TemplateID:    tpl.ID,
		TemplateName:  tpl.Name,
		Filename:      filepath.Base(filePath),
		FileSizeBytes: fileInfo.Size(),
		TotalRows:     0,
		SuccessRows:   0,
		FailedRows:    0,
		ImportedBy:    importedBy,
		StartedAt:     time.Now(),
		Status:        "IN_PROGRESS",
	}

	// Register initial import history
	_, err = s.db.Conn().Exec(`
		INSERT INTO import_history (id, template_id, filename, file_size_bytes, total_rows, success_rows, failed_rows, imported_by, started_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		history.ID, history.TemplateID, history.Filename, history.FileSizeBytes,
		0, 0, 0, history.ImportedBy, history.StartedAt, history.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mencatat riwayat import: %w", err)
	}

	tableName := template.GetTableNameForTemplate(tpl.ID)

	// Map columns to cell indexes
	type colMap struct {
		colIndex int
		column   models.TemplateColumn
	}
	var mappings []colMap
	var colNames []string
	var placeholders []string

	for _, c := range tpl.Columns {
		idx := ConvertExcelColumnNameToIndex(c.ExcelColumn)
		mappings = append(mappings, colMap{
			colIndex: idx,
			column:   c,
		})
		colNames = append(colNames, fmt.Sprintf("[%s]", c.FieldName))
		placeholders = append(placeholders, "?")
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (_import_id, %s) VALUES (?, %s)",
		tableName,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
	)

	// Cancellation setup
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.activeJobs.Store(importID, cancel)
	defer s.activeJobs.Delete(importID)

	const batchSize = 2500
	var batchArgs []interface{}
	var batchCount int64 = 0
	var errorBatch []models.ImportError

	currentRow := 0
	startTime := time.Now()
	lastProgressTime := time.Now()

	// Helper to flush batch into database transaction
	flushBatch := func() error {
		if batchCount == 0 {
			return nil
		}

		tx, err := s.db.Conn().Begin()
		if err != nil {
			return err
		}

		stmt, err := tx.Prepare(insertSQL)
		if err != nil {
			tx.Rollback()
			return err
		}
		defer stmt.Close()

		numFields := len(mappings) + 1
		for i := 0; i < int(batchCount); i++ {
			args := batchArgs[i*numFields : (i+1)*numFields]
			if _, err := stmt.Exec(args...); err != nil {
				// Record fatal row error
				_ = tx.Rollback()
				return fmt.Errorf("batch insert error: %w", err)
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		// Flush errors
		if len(errorBatch) > 0 {
			errTx, err := s.db.Conn().Begin()
			if err == nil {
				errStmt, _ := errTx.Prepare(`
					INSERT INTO import_errors (import_id, row_number, column_name, field_value, error_reason, created_at)
					VALUES (?, ?, ?, ?, ?, ?)
				`)
				if errStmt != nil {
					for _, e := range errorBatch {
						_, _ = errStmt.Exec(e.ImportID, e.RowNumber, e.ColumnName, e.FieldValue, e.ErrorReason, time.Now())
					}
					_ = errStmt.Close()
				}
				_ = errTx.Commit()
			}
			errorBatch = errorBatch[:0]
		}

		batchArgs = batchArgs[:0]
		batchCount = 0
		return nil
	}

	for rows.Next() {
		currentRow++

		select {
		case <-childCtx.Done():
			history.Status = "CANCELLED"
			_ = s.finalizeHistory(history, "Proses import dibatalkan oleh pengguna")
			return history, errors.New("import dibatalkan")
		default:
		}

		// Skip header rows
		if currentRow < tpl.DataStartRow {
			continue
		}

		rowCells, err := rows.Columns()
		if err != nil {
			continue
		}

		// Check if row is completely empty
		isEmptyRow := true
		for _, c := range rowCells {
			if strings.TrimSpace(c) != "" {
				isEmptyRow = false
				break
			}
		}
		if isEmptyRow {
			continue
		}

		history.TotalRows++
		rowValues := make([]interface{}, len(mappings))
		rowHasError := false

		for i, m := range mappings {
			rawCellVal := ""
			if m.colIndex < len(rowCells) {
				rawCellVal = rowCells[m.colIndex]
			}

			transformed, err := s.TransformAndValidate(rawCellVal, m.column)
			if err != nil {
				rowHasError = true
				errorBatch = append(errorBatch, models.ImportError{
					ImportID:    importID,
					RowNumber:   int64(currentRow),
					ColumnName:  m.column.DisplayName,
					FieldValue:  rawCellVal,
					ErrorReason: err.Error(),
				})
				break
			}
			rowValues[i] = transformed
		}

		if rowHasError {
			history.FailedRows++
		} else {
			history.SuccessRows++
			batchArgs = append(batchArgs, importID)
			batchArgs = append(batchArgs, rowValues...)
			batchCount++

			if batchCount >= batchSize {
				if err := flushBatch(); err != nil {
					history.Status = "FAILED"
					_ = s.finalizeHistory(history, err.Error())
					return history, err
				}
			}
		}

		// Notify progress periodically
		if time.Since(lastProgressTime) > 150*time.Millisecond {
			lastProgressTime = time.Now()
			elapsedSec := time.Since(startTime).Seconds()
			var speed int64 = 0
			if elapsedSec > 0 {
				speed = int64(float64(history.TotalRows) / elapsedSec)
			}

			if progressCallback != nil {
				progressCallback(models.ImportProgressEvent{
					ImportID:      importID,
					ProcessedRows: history.TotalRows,
					TotalRows:     history.TotalRows, // streaming dynamic
					SuccessRows:   history.SuccessRows,
					FailedRows:    history.FailedRows,
					Percent:       99.0, // indeterminate streaming or progress
					SpeedRPS:      speed,
					Status:        "IN_PROGRESS",
					Message:       fmt.Sprintf("Memproses baris %d (%d sukses, %d gagal)", history.TotalRows, history.SuccessRows, history.FailedRows),
				})
			}
		}
	}

	// Final flush
	if err := flushBatch(); err != nil {
		history.Status = "FAILED"
		_ = s.finalizeHistory(history, err.Error())
		return history, err
	}

	// Update dataset record count
	_ = s.db.Conn().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&tpl.RecordCount)
	_, _ = s.db.Conn().Exec("UPDATE datasets SET record_count = ?, updated_at = ? WHERE template_id = ?", tpl.RecordCount, time.Now(), tpl.ID)

	history.Status = "COMPLETED"
	_ = s.finalizeHistory(history, "")

	if progressCallback != nil {
		progressCallback(models.ImportProgressEvent{
			ImportID:      importID,
			ProcessedRows: history.TotalRows,
			TotalRows:     history.TotalRows,
			SuccessRows:   history.SuccessRows,
			FailedRows:    history.FailedRows,
			Percent:       100.0,
			Status:        "COMPLETED",
			Message:       fmt.Sprintf("Import selesai: %d data berhasil dimasukkan", history.SuccessRows),
		})
	}

	return history, nil
}

func (s *ImportService) finalizeHistory(h *models.ImportHistory, errMsg string) error {
	now := time.Now()
	h.FinishedAt = &now
	h.ErrorMessage = errMsg

	_, err := s.db.Conn().Exec(`
		UPDATE import_history SET 
			total_rows = ?, success_rows = ?, failed_rows = ?, finished_at = ?, status = ?, error_message = ?
		WHERE id = ?`,
		h.TotalRows, h.SuccessRows, h.FailedRows, h.FinishedAt, h.Status, h.ErrorMessage, h.ID,
	)
	return err
}

// CancelImport aborts an ongoing import job
func (s *ImportService) CancelImport(importID string) bool {
	if cancelFunc, ok := s.activeJobs.Load(importID); ok {
		cancelFunc.(context.CancelFunc)()
		return true
	}
	return false
}

// GetImportHistory loads import audit logs
func (s *ImportService) GetImportHistory(templateID string, limit int) ([]models.ImportHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT i.id, i.template_id, t.name, i.filename, i.file_size_bytes, i.total_rows,
		       i.success_rows, i.failed_rows, i.imported_by, i.started_at, i.finished_at, i.status, i.error_message
		FROM import_history i
		LEFT JOIN templates t ON i.template_id = t.id
	`
	var args []interface{}
	if templateID != "" {
		query += " WHERE i.template_id = ?"
		args = append(args, templateID)
	}
	query += " ORDER BY i.started_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Conn().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.ImportHistory
	for rows.Next() {
		var h models.ImportHistory
		var stStr string
		var finStr, tplName, errMsg sql.NullString

		err := rows.Scan(
			&h.ID, &h.TemplateID, &tplName, &h.Filename, &h.FileSizeBytes,
			&h.TotalRows, &h.SuccessRows, &h.FailedRows, &h.ImportedBy,
			&stStr, &finStr, &h.Status, &errMsg,
		)
		if err != nil {
			return nil, err
		}

		if tplName.Valid {
			h.TemplateName = tplName.String
		}
		if errMsg.Valid {
			h.ErrorMessage = errMsg.String
		}
		h.StartedAt, _ = time.Parse(time.RFC3339, stStr)
		if finStr.Valid {
			t, _ := time.Parse(time.RFC3339, finStr.String)
			h.FinishedAt = &t
		}

		list = append(list, h)
	}
	return list, nil
}

// GetImportErrors returns error details for a specific import
func (s *ImportService) GetImportErrors(importID string, limit int) ([]models.ImportError, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.Conn().Query(`
		SELECT id, import_id, row_number, column_name, field_value, error_reason, created_at
		FROM import_errors WHERE import_id = ? ORDER BY row_number ASC LIMIT ?
	`, importID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errs []models.ImportError
	for rows.Next() {
		var e models.ImportError
		var cr string
		var col, val sql.NullString
		if err := rows.Scan(&e.ID, &e.ImportID, &e.RowNumber, &col, &val, &e.ErrorReason, &cr); err != nil {
			return nil, err
		}
		if col.Valid {
			e.ColumnName = col.String
		}
		if val.Valid {
			e.FieldValue = val.String
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, cr)
		errs = append(errs, e)
	}
	return errs, nil
}
