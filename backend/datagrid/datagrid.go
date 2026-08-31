package datagrid

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
	"natapadu-app/backend/template"
)

type DataGridService struct {
	db          *db.Database
	templateSvc *template.TemplateService
}

func NewDataGridService(database *db.Database, tplSvc *template.TemplateService) *DataGridService {
	return &DataGridService{
		db:          database,
		templateSvc: tplSvc,
	}
}

// BuildWhereClause constructs parameterized SQL conditions securely.
// tableName dibutuhkan operator duplikat (subquery ke tabel yang sama); logic
// menentukan kondisi digabung dengan AND (default) atau OR.
func BuildWhereClause(tableName string, cols []models.TemplateColumn, filters []models.FilterCondition, searchTerm string, logic string) (string, []interface{}, error) {
	colMap := make(map[string]models.TemplateColumn)
	for _, c := range cols {
		colMap[strings.ToLower(c.FieldName)] = c
	}

	var clauses []string
	var args []interface{}

	// Global Search across string/date columns
	if strings.TrimSpace(searchTerm) != "" {
		var searchClauses []string
		term := "%" + strings.TrimSpace(searchTerm) + "%"
		for _, c := range cols {
			if strings.EqualFold(c.DataType, "STRING") || strings.EqualFold(c.DataType, "DATE") || strings.EqualFold(c.DataType, "DATETIME") {
				searchClauses = append(searchClauses, fmt.Sprintf("[%s] LIKE ?", c.FieldName))
				args = append(args, term)
			}
		}
		if len(searchClauses) > 0 {
			clauses = append(clauses, "("+strings.Join(searchClauses, " OR ")+")")
		}
	}

	// Filter Conditions
	for _, f := range filters {
		c, ok := colMap[strings.ToLower(f.FieldName)]
		if !ok {
			continue // ignore unknown columns
		}

		fieldExpr := fmt.Sprintf("[%s]", c.FieldName)
		valStr := fmt.Sprintf("%v", f.Value)

		switch strings.ToLower(f.Operator) {
		case "equals", "eq", "=":
			clauses = append(clauses, fmt.Sprintf("%s = ?", fieldExpr))
			args = append(args, f.Value)

		case "not_equals", "neq", "!=":
			clauses = append(clauses, fmt.Sprintf("%s != ?", fieldExpr))
			args = append(args, f.Value)

		case "contains", "like":
			clauses = append(clauses, fmt.Sprintf("%s LIKE ?", fieldExpr))
			args = append(args, "%"+valStr+"%")

		case "not_contains", "not_like":
			clauses = append(clauses, fmt.Sprintf("%s NOT LIKE ?", fieldExpr))
			args = append(args, "%"+valStr+"%")

		case "starts_with":
			clauses = append(clauses, fmt.Sprintf("%s LIKE ?", fieldExpr))
			args = append(args, valStr+"%")

		case "ends_with":
			clauses = append(clauses, fmt.Sprintf("%s LIKE ?", fieldExpr))
			args = append(args, "%"+valStr)

		case "is_empty", "empty":
			clauses = append(clauses, fmt.Sprintf("(%s IS NULL OR %s = '')", fieldExpr, fieldExpr))

		case "is_not_empty", "not_empty":
			clauses = append(clauses, fmt.Sprintf("(%s IS NOT NULL AND %s != '')", fieldExpr, fieldExpr))

		case "gt", ">":
			clauses = append(clauses, fmt.Sprintf("%s > ?", fieldExpr))
			args = append(args, f.Value)

		case "gte", ">=":
			clauses = append(clauses, fmt.Sprintf("%s >= ?", fieldExpr))
			args = append(args, f.Value)

		case "lt", "<":
			clauses = append(clauses, fmt.Sprintf("%s < ?", fieldExpr))
			args = append(args, f.Value)

		case "lte", "<=":
			clauses = append(clauses, fmt.Sprintf("%s <= ?", fieldExpr))
			args = append(args, f.Value)

		case "between":
			clauses = append(clauses, fmt.Sprintf("%s BETWEEN ? AND ?", fieldExpr))
			args = append(args, f.Value, f.ValueTo)

		case "before":
			clauses = append(clauses, fmt.Sprintf("%s < ?", fieldExpr))
			args = append(args, f.Value)

		case "after":
			clauses = append(clauses, fmt.Sprintf("%s > ?", fieldExpr))
			args = append(args, f.Value)

		case "in_list", "not_in_list":
			// Daftar nilai dipisah baris baru atau koma — untuk mencocokkan banyak nilai sekaligus
			items := splitValueList(valStr)
			if len(items) == 0 {
				continue
			}
			placeholders := make([]string, len(items))
			for i, it := range items {
				placeholders[i] = "?"
				args = append(args, it)
			}
			op := "IN"
			if strings.EqualFold(f.Operator, "not_in_list") {
				op = "NOT IN"
			}
			clauses = append(clauses, fmt.Sprintf("%s %s (%s)", fieldExpr, op, strings.Join(placeholders, ", ")))

		case "is_duplicate", "is_not_duplicate":
			// Nilai (opsional) berisi kolom tambahan pembentuk kunci gabungan,
			// mis. cari baris yang nama DAN perusahaannya sama-sama berulang.
			keyCols := []models.TemplateColumn{c}
			for _, extra := range splitValueList(valStr) {
				if ec, ok := colMap[strings.ToLower(extra)]; ok && !strings.EqualFold(ec.FieldName, c.FieldName) {
					keyCols = append(keyCols, ec)
				}
			}

			parts := make([]string, len(keyCols))
			groupBy := make([]string, len(keyCols))
			for i, kc := range keyCols {
				parts[i] = fmt.Sprintf("COALESCE([%s],'')", kc.FieldName)
				groupBy[i] = fmt.Sprintf("[%s]", kc.FieldName)
			}
			keyExpr := strings.Join(parts, " || CHAR(31) || ")

			op := "IN"
			if strings.EqualFold(f.Operator, "is_not_duplicate") {
				op = "NOT IN"
			}
			clauses = append(clauses, fmt.Sprintf(
				"%s %s (SELECT %s FROM %s GROUP BY %s HAVING COUNT(*) > 1)",
				keyExpr, op, keyExpr, tableName, strings.Join(groupBy, ", "),
			))
		}
	}

	whereSQL := ""
	if len(clauses) > 0 {
		joiner := " AND "
		// Pencarian global selalu dipersempit dengan AND; hanya kondisi filter yang bisa OR
		if strings.EqualFold(logic, "OR") && len(clauses) > 1 {
			searchClause := ""
			if strings.TrimSpace(searchTerm) != "" {
				searchClause = clauses[0]
				clauses = clauses[1:]
			}
			combined := "(" + strings.Join(clauses, " OR ") + ")"
			if searchClause != "" {
				combined = searchClause + " AND " + combined
			}
			return " WHERE " + combined, args, nil
		}
		whereSQL = " WHERE " + strings.Join(clauses, joiner)
	}

	return whereSQL, args, nil
}

// splitValueList memecah input multi-nilai (baris baru atau koma) menjadi daftar bersih
func splitValueList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' })
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if v := strings.TrimSpace(f); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// QueryData executes server-side paginated queries on dataset table
func (s *DataGridService) QueryData(req models.QueryRequest) (*models.QueryResponse, error) {
	start := time.Now()

	if req.TemplateID == "" {
		return nil, errors.New("template ID is required")
	}

	tpl, err := s.templateSvc.GetTemplateByID(req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template tidak ditemukan: %w", err)
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	if req.PageSize > 500 {
		req.PageSize = 500
	}

	tableName := template.GetTableNameForTemplate(tpl.ID)
	whereSQL, args, err := BuildWhereClause(tableName, tpl.Columns, req.Filters, req.SearchTerm, req.FilterLogic)
	if err != nil {
		return nil, err
	}

	// 1. Total Count Query
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", tableName, whereSQL)
	var totalRows int64
	if err := s.db.Conn().QueryRow(countSQL, args...).Scan(&totalRows); err != nil {
		return nil, fmt.Errorf("gagal menghitung total data: %w", err)
	}

	// 2. Build Sort
	sortCol := "_row_id"
	sortDir := "ASC"
	if req.SortBy != "" {
		// Verify sort column is valid
		for _, c := range tpl.Columns {
			if strings.EqualFold(c.FieldName, req.SortBy) {
				sortCol = fmt.Sprintf("[%s]", c.FieldName)
				break
			}
		}
	}
	if strings.EqualFold(req.SortOrder, "DESC") {
		sortDir = "DESC"
	}

	// 3. Build Select Columns
	var selectCols []string
	selectCols = append(selectCols, "_row_id", "_import_id", "_created_at")
	for _, c := range tpl.Columns {
		selectCols = append(selectCols, fmt.Sprintf("[%s]", c.FieldName))
	}

	offset := (req.Page - 1) * req.PageSize
	querySQL := fmt.Sprintf(
		"SELECT %s FROM %s%s ORDER BY %s %s LIMIT ? OFFSET ?",
		strings.Join(selectCols, ", "),
		tableName,
		whereSQL,
		sortCol,
		sortDir,
	)

	queryArgs := append(args, req.PageSize, offset)
	rows, err := s.db.Conn().Query(querySQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("gagal query data: %w", err)
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		colValues := make([]interface{}, len(colNames))
		colValuePointers := make([]interface{}, len(colNames))
		for i := range colValues {
			colValuePointers[i] = &colValues[i]
		}

		if err := rows.Scan(colValuePointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, colName := range colNames {
			val := colValues[i]
			// Handle []byte strings from SQLite driver
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	totalPages := int((totalRows + int64(req.PageSize) - 1) / int64(req.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.QueryResponse{
		Data:        results,
		TotalRows:   totalRows,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  totalPages,
		Columns:     tpl.Columns,
		ExecutionMs: time.Since(start).Milliseconds(),
	}, nil
}

// DeleteRow deletes a single row by _row_id
func (s *DataGridService) DeleteRow(templateID string, rowID int64) error {
	tableName := template.GetTableNameForTemplate(templateID)
	res, err := s.db.Conn().Exec(fmt.Sprintf("DELETE FROM %s WHERE _row_id = ?", tableName), rowID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("baris data tidak ditemukan")
	}
	return nil
}

// BulkDeleteRows deletes multiple rows
func (s *DataGridService) BulkDeleteRows(templateID string, rowIDs []int64) (int64, error) {
	if len(rowIDs) == 0 {
		return 0, nil
	}

	tableName := template.GetTableNameForTemplate(templateID)
	placeholders := make([]string, len(rowIDs))
	args := make([]interface{}, len(rowIDs))
	for i, id := range rowIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE _row_id IN (%s)", tableName, strings.Join(placeholders, ", "))
	res, err := s.db.Conn().Exec(query, args...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

// TruncateDataset removes all records in a template dataset
func (s *DataGridService) TruncateDataset(templateID string) error {
	tableName := template.GetTableNameForTemplate(templateID)
	_, err := s.db.Conn().Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	if err != nil {
		return err
	}
	_, _ = s.db.Conn().Exec("UPDATE datasets SET record_count = 0, updated_at = ? WHERE template_id = ?", time.Now(), templateID)
	return nil
}

// UpsertRow menambah (rowID <= 0) atau memperbarui satu baris dataset secara manual.
// Hanya field yang terdaftar di template yang diterima, jadi nama kolom tidak pernah
// berasal dari input pengguna. Nilai sudah lolos transform+validasi di lapisan pemanggil.
func (s *DataGridService) UpsertRow(templateID string, rowID int64, values map[string]interface{}) (int64, error) {
	tpl, err := s.templateSvc.GetTemplateByID(templateID)
	if err != nil {
		return 0, fmt.Errorf("workspace tidak ditemukan: %w", err)
	}
	if len(tpl.Columns) == 0 {
		return 0, errors.New("workspace belum punya definisi kolom")
	}

	tableName := template.GetTableNameForTemplate(templateID)

	var fields []string
	var args []interface{}
	for _, c := range tpl.Columns {
		v, ok := values[c.FieldName]
		if !ok {
			continue
		}
		fields = append(fields, c.FieldName)
		args = append(args, v)
	}
	if len(fields) == 0 {
		return 0, errors.New("tidak ada kolom yang diisi")
	}

	if rowID > 0 {
		sets := make([]string, 0, len(fields)+1)
		for _, f := range fields {
			sets = append(sets, fmt.Sprintf("[%s] = ?", f))
		}
		sets = append(sets, "_updated_at = ?")
		args = append(args, time.Now())
		args = append(args, rowID)

		query := fmt.Sprintf("UPDATE %s SET %s WHERE _row_id = ?", tableName, strings.Join(sets, ", "))
		res, err := s.db.Conn().Exec(query, args...)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return 0, errors.New("baris data tidak ditemukan")
		}
		return rowID, nil
	}

	cols := make([]string, 0, len(fields)+1)
	placeholders := make([]string, 0, len(fields)+1)
	for _, f := range fields {
		cols = append(cols, fmt.Sprintf("[%s]", f))
		placeholders = append(placeholders, "?")
	}
	cols = append(cols, "_import_id")
	placeholders = append(placeholders, "?")
	args = append(args, "MANUAL")

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	res, err := s.db.Conn().Exec(query, args...)
	if err != nil {
		return 0, err
	}
	newID, _ := res.LastInsertId()

	_, _ = s.db.Conn().Exec(
		"UPDATE datasets SET record_count = record_count + 1, updated_at = ? WHERE template_id = ?",
		time.Now(), templateID,
	)
	return newID, nil
}

// GetRow mengambil satu baris untuk diedit manual
func (s *DataGridService) GetRow(templateID string, rowID int64) (map[string]interface{}, error) {
	tpl, err := s.templateSvc.GetTemplateByID(templateID)
	if err != nil {
		return nil, fmt.Errorf("workspace tidak ditemukan: %w", err)
	}

	tableName := template.GetTableNameForTemplate(templateID)
	cols := make([]string, 0, len(tpl.Columns))
	for _, c := range tpl.Columns {
		cols = append(cols, fmt.Sprintf("[%s]", c.FieldName))
	}
	if len(cols) == 0 {
		return nil, errors.New("workspace belum punya definisi kolom")
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE _row_id = ?", strings.Join(cols, ", "), tableName)
	vals := make([]interface{}, len(tpl.Columns))
	ptrs := make([]interface{}, len(tpl.Columns))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := s.db.Conn().QueryRow(query, rowID).Scan(ptrs...); err != nil {
		return nil, err
	}

	out := make(map[string]interface{}, len(tpl.Columns))
	for i, c := range tpl.Columns {
		if b, ok := vals[i].([]byte); ok {
			out[c.FieldName] = string(b)
		} else {
			out[c.FieldName] = vals[i]
		}
	}
	return out, nil
}

// GetDistinctValues mengembalikan nilai unik sebuah kolom beserta jumlah barisnya,
// dipakai untuk menelusuri data per kelompok (mis. satu per satu tiap perusahaan).
// fieldName divalidasi terhadap definisi template, jadi tidak pernah masuk SQL apa adanya.
func (s *DataGridService) GetDistinctValues(templateID, fieldName, search string, limit int) ([]models.DistinctValue, error) {
	tpl, err := s.templateSvc.GetTemplateByID(templateID)
	if err != nil {
		return nil, fmt.Errorf("workspace tidak ditemukan: %w", err)
	}

	var field string
	for _, c := range tpl.Columns {
		if strings.EqualFold(c.FieldName, fieldName) {
			field = c.FieldName
			break
		}
	}
	if field == "" {
		return nil, fmt.Errorf("kolom '%s' tidak ada di workspace ini", fieldName)
	}

	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	tableName := template.GetTableNameForTemplate(templateID)
	query := fmt.Sprintf("SELECT [%s] AS v, COUNT(*) AS n FROM %s", field, tableName)
	var args []interface{}
	if strings.TrimSpace(search) != "" {
		query += fmt.Sprintf(" WHERE [%s] LIKE ?", field)
		args = append(args, "%"+search+"%")
	}
	query += fmt.Sprintf(" GROUP BY [%s] ORDER BY n DESC, v ASC LIMIT ?", field)
	args = append(args, limit)

	rows, err := s.db.Conn().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil nilai unik: %w", err)
	}
	defer rows.Close()

	out := make([]models.DistinctValue, 0)
	for rows.Next() {
		var v sql.NullString
		var n int64
		if err := rows.Scan(&v, &n); err != nil {
			continue
		}
		out = append(out, models.DistinctValue{Value: v.String, Count: n})
	}
	return out, nil
}

// GetDuplicateGroups mencari kombinasi nilai yang muncul lebih dari sekali.
// fields boleh lebih dari satu kolom, mis. cari baris yang nama DAN tanggal lahirnya sama.
func (s *DataGridService) GetDuplicateGroups(templateID string, fields []string, search string, limit int) ([]models.DuplicateGroup, error) {
	tpl, err := s.templateSvc.GetTemplateByID(templateID)
	if err != nil {
		return nil, fmt.Errorf("workspace tidak ditemukan: %w", err)
	}

	// Hanya kolom yang terdaftar di template yang boleh masuk SQL
	colMap := make(map[string]models.TemplateColumn, len(tpl.Columns))
	for _, c := range tpl.Columns {
		colMap[strings.ToLower(c.FieldName)] = c
	}
	var keys []models.TemplateColumn
	for _, f := range fields {
		if c, ok := colMap[strings.ToLower(f)]; ok {
			keys = append(keys, c)
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("pilih minimal satu kolom pembanding")
	}

	if limit <= 0 || limit > 2000 {
		limit = 500
	}

	selectCols := make([]string, len(keys))
	groupCols := make([]string, len(keys))
	for i, c := range keys {
		selectCols[i] = fmt.Sprintf("COALESCE([%s],'')", c.FieldName)
		groupCols[i] = fmt.Sprintf("[%s]", c.FieldName)
	}

	tableName := template.GetTableNameForTemplate(templateID)
	query := fmt.Sprintf("SELECT %s, COUNT(*) AS n FROM %s", strings.Join(selectCols, ", "), tableName)

	var args []interface{}
	if strings.TrimSpace(search) != "" {
		var likes []string
		for _, c := range keys {
			likes = append(likes, fmt.Sprintf("[%s] LIKE ?", c.FieldName))
			args = append(args, "%"+strings.TrimSpace(search)+"%")
		}
		query += " WHERE (" + strings.Join(likes, " OR ") + ")"
	}
	query += fmt.Sprintf(" GROUP BY %s HAVING n > 1 ORDER BY n DESC LIMIT ?", strings.Join(groupCols, ", "))
	args = append(args, limit)

	rows, err := s.db.Conn().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal mencari duplikat: %w", err)
	}
	defer rows.Close()

	out := make([]models.DuplicateGroup, 0)
	for rows.Next() {
		vals := make([]interface{}, len(keys)+1)
		ptrs := make([]interface{}, len(keys)+1)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}

		g := models.DuplicateGroup{Values: make([]string, len(keys))}
		for i := 0; i < len(keys); i++ {
			switch v := vals[i].(type) {
			case nil:
				g.Values[i] = ""
			case []byte:
				g.Values[i] = string(v)
			default:
				g.Values[i] = fmt.Sprint(v)
			}
		}
		if n, ok := vals[len(keys)].(int64); ok {
			g.Count = n
		}
		out = append(out, g)
	}
	return out, nil
}
