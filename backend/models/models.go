package models

import (
	"time"
)

// User represents an authenticated local operator/admin
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"displayName"`
	Role         string    `json:"role"`   // 'ADMIN' or 'USER'
	Status       string    `json:"status"` // 'ACTIVE' or 'INACTIVE'
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// UserSession is returned upon successful authentication
type UserSession struct {
	User      User      `json:"user"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Template represents a customizable Excel mapping configuration
type Template struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	SheetName    string           `json:"sheetName"`
	HeaderRow    int              `json:"headerRow"`
	DataStartRow int              `json:"dataStartRow"`
	Version      int              `json:"version"`
	Status       string           `json:"status"` // 'ACTIVE' or 'ARCHIVED'
	Columns      []TemplateColumn `json:"columns"`
	RecordCount  int64            `json:"recordCount"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// TemplateColumn defines mapping, types, transforms, and validations for a column
type TemplateColumn struct {
	ID              string `json:"id"`
	TemplateID      string `json:"templateId"`
	ExcelColumn     string `json:"excelColumn"`   // e.g. "A", "B", "C"
	FieldName       string `json:"fieldName"`     // Sanitized SQLite column name, e.g. "nik", "nama"
	DisplayName     string `json:"displayName"`   // Friendly UI label, e.g. "NIK Nasabah"
	DataType        string `json:"dataType"`      // STRING, INTEGER, DECIMAL, BOOLEAN, DATE, DATETIME, CURRENCY, PERCENTAGE
	FormatPattern   string `json:"formatPattern"` // e.g. "DD/MM/YYYY", "YYYY-MM-DD", "Rp #"
	Required        bool   `json:"required"`
	IsUnique        bool   `json:"isUnique"`
	DefaultValue    string `json:"defaultValue"`
	TransformRules  string `json:"transformRules"`  // JSON array of strings, e.g. ["TRIM", "UPPERCASE"]
	ValidationRules string `json:"validationRules"` // JSON object, e.g. {"minLength": 16, "maxLength": 16, "regex": "^[0-9]+$"}
	SortOrder       int    `json:"sortOrder"`
	IsIndexed       bool   `json:"isIndexed"`
}

// Dataset represents metadata of physical SQLite table created for a template
type Dataset struct {
	ID          string    `json:"id"`
	TemplateID  string    `json:"templateId"`
	TableName   string    `json:"tableName"`
	RecordCount int64     `json:"recordCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ImportHistory tracks audit trail of data ingestions
type ImportHistory struct {
	ID            string     `json:"id"`
	TemplateID    string     `json:"templateId"`
	TemplateName  string     `json:"templateName,omitempty"`
	Filename      string     `json:"filename"`
	FileSizeBytes int64      `json:"fileSizeBytes"`
	TotalRows     int64      `json:"totalRows"`
	SuccessRows   int64      `json:"successRows"`
	FailedRows    int64      `json:"failedRows"`
	ImportedBy    string     `json:"importedBy"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	Status        string     `json:"status"` // 'IN_PROGRESS', 'COMPLETED', 'FAILED', 'CANCELLED'
	ErrorMessage  string     `json:"errorMessage,omitempty"`
}

// ImportError tracks details of rejected rows
type ImportError struct {
	ID          int64     `json:"id"`
	ImportID    string    `json:"importId"`
	RowNumber   int64     `json:"rowNumber"`
	ColumnName  string    `json:"columnName"`
	FieldValue  string    `json:"fieldValue"`
	ErrorReason string    `json:"errorReason"`
	CreatedAt   time.Time `json:"createdAt"`
}

// FilterCondition represents a single condition in visual filter builder
type FilterCondition struct {
	FieldName string      `json:"fieldName"`
	Operator  string      `json:"operator"` // equals, not_equals, contains, not_contains, starts_with, ends_with, is_empty, is_not_empty, gt, gte, lt, lte, between, before, after
	Value     interface{} `json:"value"`
	ValueTo   interface{} `json:"valueTo,omitempty"` // For 'between'
}

// ChartPoint adalah satu titik/batang pada visualisasi dashboard
type ChartPoint struct {
	Label     string `json:"label"`
	Value     int64  `json:"value"`
	Secondary int64  `json:"secondary"`
}

// RowError menunjuk satu sel yang gagal validasi saat penyimpanan massal
type RowError struct {
	Index  int    `json:"index"`  // baris ke berapa pada masukan (0-based)
	Field  string `json:"field"`  // nama kolom database
	Column string `json:"column"` // label kolom untuk ditampilkan
	Reason string `json:"reason"`
}

// SaveRowsResult adalah hasil penyimpanan banyak baris sekaligus
type SaveRowsResult struct {
	Saved   int64      `json:"saved"`
	Skipped int        `json:"skipped"` // baris kosong yang diabaikan
	Errors  []RowError `json:"errors"`
}

// DuplicateGroup adalah satu kombinasi nilai yang muncul lebih dari sekali
type DuplicateGroup struct {
	Values []string `json:"values"`
	Count  int64    `json:"count"`
}

// QueryRequest defines server-side pagination, searching, sorting, and filtering
type QueryRequest struct {
	TemplateID  string            `json:"templateId"`
	Page        int               `json:"page"`       // 1-indexed
	PageSize    int               `json:"pageSize"`   // Default 50, max 500
	SearchTerm  string            `json:"searchTerm"` // Global search across text columns
	SortBy      string            `json:"sortBy"`     // Field name
	SortOrder   string            `json:"sortOrder"`  // 'ASC' or 'DESC'
	Filters     []FilterCondition `json:"filters"`
	FilterLogic string            `json:"filterLogic"` // 'AND' (default) atau 'OR'
}

// QueryResponse delivers paginated data results
type QueryResponse struct {
	Data        []map[string]interface{} `json:"data"`
	TotalRows   int64                    `json:"totalRows"`
	Page        int                      `json:"page"`
	PageSize    int                      `json:"pageSize"`
	TotalPages  int                      `json:"totalPages"`
	Columns     []TemplateColumn         `json:"columns"`
	ExecutionMs int64                    `json:"executionMs"`
}

// SavedFilter stores reusable filter presets
type SavedFilter struct {
	ID            string    `json:"id"`
	TemplateID    string    `json:"templateId"`
	Name          string    `json:"name"`
	FilterPayload string    `json:"filterPayload"` // JSON of []FilterCondition
	CreatedBy     string    `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ActivityLog stores system operational logs
type ActivityLog struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ipAddress"`
	CreatedAt time.Time `json:"createdAt"`
}

// AppSummary represents metrics for dashboard
type AppSummary struct {
	TotalTemplates int64           `json:"totalTemplates"`
	TotalRecords   int64           `json:"totalRecords"`
	TotalImports   int64           `json:"totalImports"`
	TotalUsers     int64           `json:"totalUsers"`
	DatabaseSize   string          `json:"databaseSize"`
	SuccessRows    int64           `json:"successRows"`
	FailedRows     int64           `json:"failedRows"`
	RecentImports  []ImportHistory `json:"recentImports"`
	RecentActivity []ActivityLog   `json:"recentActivity"`

	// Agregat untuk visualisasi dashboard
	ImportTrend       []ChartPoint `json:"importTrend"`       // 14 hari: value=sukses, secondary=gagal
	WorkspaceSizes    []ChartPoint `json:"workspaceSizes"`    // baris per workspace
	ActivityBreakdown []ChartPoint `json:"activityBreakdown"` // jumlah aktivitas per jenis
}

// ImportProgressEvent is sent via event bus to UI
type ImportProgressEvent struct {
	ImportID      string  `json:"importId"`
	ProcessedRows int64   `json:"processedRows"`
	TotalRows     int64   `json:"totalRows"`
	SuccessRows   int64   `json:"successRows"`
	FailedRows    int64   `json:"failedRows"`
	Percent       float64 `json:"percent"`
	SpeedRPS      int64   `json:"speedRps"`
	Status        string  `json:"status"`
	Message       string  `json:"message"`
}

// ExportRequest specifies export options
type ExportRequest struct {
	TemplateID     string            `json:"templateId"`
	Format         string            `json:"format"` // 'XLSX', 'CSV', 'ODS' (default XLSX)
	Scope          string            `json:"scope"`  // 'ALL', 'FILTERED', 'SELECTED'
	SelectedRowIDs []int64           `json:"selectedRowIds,omitempty"`
	Columns        []string          `json:"columns"` // Field names to export
	SearchTerm     string            `json:"searchTerm,omitempty"`
	Filters        []FilterCondition `json:"filters,omitempty"`
	SortBy         string            `json:"sortBy,omitempty"`
	SortOrder      string            `json:"sortOrder,omitempty"`
	FilterLogic    string            `json:"filterLogic,omitempty"` // 'AND' (default) atau 'OR'
	OutputFilename string            `json:"outputFilename"`
}

// ExportResult adalah hasil satu operasi export.
// Dibungkus struct karena binding Wails hanya mendukung dua nilai balik (value, error) —
// mengembalikan (string, int64, error) membuat binding-nya menghasilkan null di sisi frontend.
type ExportResult struct {
	FilePath string `json:"filePath"`
	RowCount int64  `json:"rowCount"`
	Format   string `json:"format"`
}

// DistinctValue adalah satu nilai unik pada sebuah kolom beserta jumlah barisnya
type DistinctValue struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// ExcelSheetPreview provides sample parsed sheet headers & rows
type ExcelSheetPreview struct {
	Sheets       []string   `json:"sheets"`
	ActiveSheet  string     `json:"activeSheet"`
	TotalRows    int        `json:"totalRows"`
	TotalColumns int        `json:"totalColumns"`
	HeaderRow    int        `json:"headerRow"`
	DataStartRow int        `json:"dataStartRow"`
	Headers      []string   `json:"headers"`
	SampleRows   [][]string `json:"sampleRows"`
}
