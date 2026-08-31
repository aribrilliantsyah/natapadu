package template

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"

	"github.com/google/uuid"
)

type TemplateService struct {
	db *db.Database
}

func NewTemplateService(database *db.Database) *TemplateService {
	return &TemplateService{db: database}
}

var safeNameRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// SanitizeIdentifier converts any display string into a safe SQL identifier
func SanitizeIdentifier(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = safeNameRegex.ReplaceAllString(s, "_")
	// Collapse multiple underscores
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	if s == "" {
		s = "field"
	}
	// Avoid reserved SQL words prefixing if needed
	return s
}

// GetTableNameForTemplate returns the physical dataset table name for a template ID
func GetTableNameForTemplate(templateID string) string {
	sanitized := strings.ReplaceAll(templateID, "-", "")
	return fmt.Sprintf("dataset_%s", sanitized)
}

// GetAllTemplates retrieves all templates with their column definitions
func (s *TemplateService) GetAllTemplates() ([]models.Template, error) {
	list := make([]models.Template, 0)
	rows, err := s.db.Conn().Query(`
		SELECT id, name, description, sheet_name, header_row, data_start_row, version, status, created_at, updated_at
		FROM templates WHERE status != 'ARCHIVED' ORDER BY created_at DESC
	`)
	if err != nil {
		return list, nil
	}

	// Baca habis lalu tutup rows SEBELUM query turunan apa pun: pool SQLite hanya
	// punya 1 koneksi, jadi query bersarang di dalam loop ini akan deadlock.
	for rows.Next() {
		var t models.Template
		var desc sql.NullString
		var cr, up string
		if err := rows.Scan(&t.ID, &t.Name, &desc, &t.SheetName, &t.HeaderRow, &t.DataStartRow, &t.Version, &t.Status, &cr, &up); err != nil {
			continue
		}
		if desc.Valid {
			t.Description = desc.String
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, cr)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, up)
		list = append(list, t)
	}
	rows.Close()

	for i := range list {
		tblName := GetTableNameForTemplate(list[i].ID)
		_ = s.db.Conn().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tblName)).Scan(&list[i].RecordCount)

		cols, _ := s.GetTemplateColumns(list[i].ID)
		if cols == nil {
			cols = make([]models.TemplateColumn, 0)
		}
		list[i].Columns = cols
	}

	return list, nil
}

// GetTemplateByID fetches a template by ID
func (s *TemplateService) GetTemplateByID(id string) (*models.Template, error) {
	var t models.Template
	var desc sql.NullString
	var cr, up string

	err := s.db.Conn().QueryRow(`
		SELECT id, name, description, sheet_name, header_row, data_start_row, version, status, created_at, updated_at
		FROM templates WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &desc, &t.SheetName, &t.HeaderRow, &t.DataStartRow, &t.Version, &t.Status, &cr, &up)

	if err != nil {
		return nil, err
	}
	if desc.Valid {
		t.Description = desc.String
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, cr)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, up)

	tblName := GetTableNameForTemplate(t.ID)
	_ = s.db.Conn().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tblName)).Scan(&t.RecordCount)

	cols, err := s.GetTemplateColumns(t.ID)
	if err != nil {
		return nil, err
	}
	t.Columns = cols

	return &t, nil
}

// GetTemplateColumns loads column definitions for a template
func (s *TemplateService) GetTemplateColumns(templateID string) ([]models.TemplateColumn, error) {
	rows, err := s.db.Conn().Query(`
		SELECT id, template_id, excel_column, field_name, display_name, data_type, 
		       format_pattern, required, is_unique, default_value, transform_rules, 
		       validation_rules, sort_order, is_indexed
		FROM template_columns WHERE template_id = ? ORDER BY sort_order ASC, excel_column ASC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []models.TemplateColumn
	for rows.Next() {
		var c models.TemplateColumn
		var fmtPat, defVal, transRules, valRules sql.NullString
		var req, unq, idx int

		err := rows.Scan(
			&c.ID, &c.TemplateID, &c.ExcelColumn, &c.FieldName, &c.DisplayName,
			&c.DataType, &fmtPat, &req, &unq, &defVal,
			&transRules, &valRules, &c.SortOrder, &idx,
		)
		if err != nil {
			return nil, err
		}

		c.Required = req == 1
		c.IsUnique = unq == 1
		c.IsIndexed = idx == 1
		if fmtPat.Valid {
			c.FormatPattern = fmtPat.String
		}
		if defVal.Valid {
			c.DefaultValue = defVal.String
		}
		if transRules.Valid {
			c.TransformRules = transRules.String
		}
		if valRules.Valid {
			c.ValidationRules = valRules.String
		}

		cols = append(cols, c)
	}
	return cols, nil
}

// CreateTemplate creates metadata and provisions dedicated physical SQLite table
func (s *TemplateService) CreateTemplate(t *models.Template) (*models.Template, error) {
	if t.Name == "" {
		return nil, errors.New("nama template tidak boleh kosong")
	}
	if len(t.Columns) == 0 {
		return nil, errors.New("minimal harus ada 1 kolom pada template")
	}

	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.SheetName == "" {
		t.SheetName = "Sheet1"
	}
	if t.HeaderRow <= 0 {
		t.HeaderRow = 1
	}
	if t.DataStartRow <= 0 {
		t.DataStartRow = 2
	}
	t.Version = 1
	t.Status = "ACTIVE"
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	// Sanitize field names and prevent duplicates
	seenFields := make(map[string]bool)
	for i := range t.Columns {
		if t.Columns[i].ID == "" {
			t.Columns[i].ID = uuid.New().String()
		}
		t.Columns[i].TemplateID = t.ID
		if t.Columns[i].FieldName == "" {
			t.Columns[i].FieldName = SanitizeIdentifier(t.Columns[i].DisplayName)
		} else {
			t.Columns[i].FieldName = SanitizeIdentifier(t.Columns[i].FieldName)
		}
		// Ensure unique field name in table
		base := t.Columns[i].FieldName
		counter := 1
		for seenFields[t.Columns[i].FieldName] {
			t.Columns[i].FieldName = fmt.Sprintf("%s_%d", base, counter)
			counter++
		}
		seenFields[t.Columns[i].FieldName] = true

		if t.Columns[i].DataType == "" {
			t.Columns[i].DataType = "STRING"
		}
		if t.Columns[i].SortOrder == 0 {
			t.Columns[i].SortOrder = i + 1
		}
	}

	tx, err := s.db.Conn().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Insert Template
	_, err = tx.Exec(`
		INSERT INTO templates (id, name, description, sheet_name, header_row, data_start_row, version, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Description, t.SheetName, t.HeaderRow, t.DataStartRow, t.Version, t.Status, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert template: %w", err)
	}

	// 2. Insert Columns
	for _, col := range t.Columns {
		reqVal := 0
		if col.Required {
			reqVal = 1
		}
		unqVal := 0
		if col.IsUnique {
			unqVal = 1
		}
		idxVal := 0
		if col.IsIndexed {
			idxVal = 1
		}

		_, err = tx.Exec(`
			INSERT INTO template_columns (
				id, template_id, excel_column, field_name, display_name, data_type,
				format_pattern, required, is_unique, default_value, transform_rules,
				validation_rules, sort_order, is_indexed
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			col.ID, col.TemplateID, col.ExcelColumn, col.FieldName, col.DisplayName, col.DataType,
			col.FormatPattern, reqVal, unqVal, col.DefaultValue, col.TransformRules,
			col.ValidationRules, col.SortOrder, idxVal,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert column '%s': %w", col.FieldName, err)
		}
	}

	// 3. Register Dataset
	tableName := GetTableNameForTemplate(t.ID)
	_, err = tx.Exec(`
		INSERT INTO datasets (id, template_id, table_name, record_count, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?)`,
		uuid.New().String(), t.ID, tableName, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register dataset: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// 4. Provision Physical SQLite Table
	if err := s.createPhysicalTable(tableName, t.Columns); err != nil {
		return nil, fmt.Errorf("failed to provision table %s: %w", tableName, err)
	}

	return t, nil
}

// UpdateTemplate updates template metadata and alters/provisions physical table
func (s *TemplateService) UpdateTemplate(t *models.Template) (*models.Template, error) {
	if t.ID == "" {
		return nil, errors.New("template ID is required")
	}

	existing, err := s.GetTemplateByID(t.ID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	t.Version = existing.Version + 1
	t.UpdatedAt = time.Now()

	tx, err := s.db.Conn().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE templates SET name = ?, description = ?, sheet_name = ?, header_row = ?, data_start_row = ?, version = ?, updated_at = ?
		WHERE id = ?`,
		t.Name, t.Description, t.SheetName, t.HeaderRow, t.DataStartRow, t.Version, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return nil, err
	}

	// Delete old columns and re-insert
	_, err = tx.Exec("DELETE FROM template_columns WHERE template_id = ?", t.ID)
	if err != nil {
		return nil, err
	}

	seenFields := make(map[string]bool)
	for i := range t.Columns {
		if t.Columns[i].ID == "" {
			t.Columns[i].ID = uuid.New().String()
		}
		t.Columns[i].TemplateID = t.ID
		t.Columns[i].FieldName = SanitizeIdentifier(t.Columns[i].FieldName)
		base := t.Columns[i].FieldName
		counter := 1
		for seenFields[t.Columns[i].FieldName] {
			t.Columns[i].FieldName = fmt.Sprintf("%s_%d", base, counter)
			counter++
		}
		seenFields[t.Columns[i].FieldName] = true

		reqVal := 0
		if t.Columns[i].Required {
			reqVal = 1
		}
		unqVal := 0
		if t.Columns[i].IsUnique {
			unqVal = 1
		}
		idxVal := 0
		if t.Columns[i].IsIndexed {
			idxVal = 1
		}

		_, err = tx.Exec(`
			INSERT INTO template_columns (
				id, template_id, excel_column, field_name, display_name, data_type,
				format_pattern, required, is_unique, default_value, transform_rules,
				validation_rules, sort_order, is_indexed
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.Columns[i].ID, t.Columns[i].TemplateID, t.Columns[i].ExcelColumn, t.Columns[i].FieldName, t.Columns[i].DisplayName, t.Columns[i].DataType,
			t.Columns[i].FormatPattern, reqVal, unqVal, t.Columns[i].DefaultValue, t.Columns[i].TransformRules,
			t.Columns[i].ValidationRules, t.Columns[i].SortOrder, idxVal,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Synchronize table columns (adding missing columns)
	tableName := GetTableNameForTemplate(t.ID)
	_ = s.syncPhysicalTableColumns(tableName, t.Columns)

	return t, nil
}

// DeleteTemplate marks template archived and drops physical table
func (s *TemplateService) DeleteTemplate(id string) error {
	tableName := GetTableNameForTemplate(id)
	_, _ = s.db.Conn().Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	_, err := s.db.Conn().Exec("DELETE FROM templates WHERE id = ?", id)
	return err
}

// DuplicateTemplate creates a clone of an existing template
func (s *TemplateService) DuplicateTemplate(sourceID string, newName string) (*models.Template, error) {
	source, err := s.GetTemplateByID(sourceID)
	if err != nil {
		return nil, err
	}

	cloned := *source
	cloned.ID = uuid.New().String()
	cloned.Name = newName
	if cloned.Name == "" {
		cloned.Name = fmt.Sprintf("%s (Copy)", source.Name)
	}

	var newCols []models.TemplateColumn
	for _, c := range source.Columns {
		colCopy := c
		colCopy.ID = uuid.New().String()
		colCopy.TemplateID = cloned.ID
		newCols = append(newCols, colCopy)
	}
	cloned.Columns = newCols

	return s.CreateTemplate(&cloned)
}

func (s *TemplateService) createPhysicalTable(tableName string, cols []models.TemplateColumn) error {
	var colDefs []string
	colDefs = append(colDefs, "_row_id INTEGER PRIMARY KEY AUTOINCREMENT")
	colDefs = append(colDefs, "_import_id TEXT NOT NULL")
	colDefs = append(colDefs, "_created_at DATETIME DEFAULT CURRENT_TIMESTAMP")
	colDefs = append(colDefs, "_updated_at DATETIME DEFAULT CURRENT_TIMESTAMP")

	for _, col := range cols {
		sqlType := "TEXT"
		switch strings.ToUpper(col.DataType) {
		case "INTEGER":
			sqlType = "INTEGER"
		case "DECIMAL", "CURRENCY", "PERCENTAGE":
			sqlType = "REAL"
		case "BOOLEAN":
			sqlType = "INTEGER"
		default:
			sqlType = "TEXT"
		}
		colDefs = append(colDefs, fmt.Sprintf("[%s] %s", col.FieldName, sqlType))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(colDefs, ", "))
	if _, err := s.db.Conn().Exec(query); err != nil {
		return err
	}

	// Create Indexes
	for _, col := range cols {
		if col.IsIndexed {
			idxName := fmt.Sprintf("idx_%s_%s", tableName, col.FieldName)
			_ = s.createIndex(tableName, col.FieldName, idxName)
		}
	}

	return nil
}

func (s *TemplateService) syncPhysicalTableColumns(tableName string, cols []models.TemplateColumn) error {
	// First ensure table exists
	_ = s.createPhysicalTable(tableName, cols)

	// Fetch existing table columns
	rows, err := s.db.Conn().Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
			existing[strings.ToLower(name)] = true
		}
	}
	rows.Close() // lepaskan koneksi tunggal sebelum ALTER TABLE di bawah

	for _, col := range cols {
		if !existing[strings.ToLower(col.FieldName)] {
			sqlType := "TEXT"
			switch strings.ToUpper(col.DataType) {
			case "INTEGER":
				sqlType = "INTEGER"
			case "DECIMAL", "CURRENCY", "PERCENTAGE":
				sqlType = "REAL"
			case "BOOLEAN":
				sqlType = "INTEGER"
			}
			alterQ := fmt.Sprintf("ALTER TABLE %s ADD COLUMN [%s] %s", tableName, col.FieldName, sqlType)
			_, _ = s.db.Conn().Exec(alterQ)
		}
	}

	return nil
}

func (s *TemplateService) createIndex(tableName, fieldName, indexName string) error {
	idxQ := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s ([%s])", indexName, tableName, fieldName)
	_, err := s.db.Conn().Exec(idxQ)
	return err
}

// ExportTemplateSchema exports a template definition to a JSON string
func (s *TemplateService) ExportTemplateSchema(templateID string) (string, error) {
	t, err := s.GetTemplateByID(templateID)
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ImportTemplateSchema parses and saves template definition from JSON string
func (s *TemplateService) ImportTemplateSchema(jsonStr string) (*models.Template, error) {
	var t models.Template
	if err := json.Unmarshal([]byte(jsonStr), &t); err != nil {
		return nil, fmt.Errorf("format JSON template tidak valid: %w", err)
	}
	t.ID = uuid.New().String()
	t.Name = fmt.Sprintf("%s (Imported)", t.Name)
	return s.CreateTemplate(&t)
}
