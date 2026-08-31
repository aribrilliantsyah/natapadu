package activity

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
)

type ActivityService struct {
	db *db.Database
}

func NewActivityService(database *db.Database) *ActivityService {
	return &ActivityService{db: database}
}

// Log records an audit activity entry
func (s *ActivityService) Log(userID, username, action, target, details string) error {
	_, err := s.db.Conn().Exec(`
		INSERT INTO activity_logs (user_id, username, action, target, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, username, action, target, details, time.Now(),
	)
	return err
}

// GetRecentLogs returns the latest activity log records
func (s *ActivityService) GetRecentLogs(limit int) ([]models.ActivityLog, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Conn().Query(`
		SELECT id, user_id, username, action, target, details, ip_address, created_at
		FROM activity_logs ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.ActivityLog
	for rows.Next() {
		var l models.ActivityLog
		var cr string
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Action, &l.Target, &l.Details, &l.IPAddress, &cr); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, cr)
		logs = append(logs, l)
	}
	return logs, nil
}

// GetDashboardSummary aggregates statistics for the dashboard
func (s *ActivityService) GetDashboardSummary() (*models.AppSummary, error) {
	summary := &models.AppSummary{}

	// Total Templates
	_ = s.db.Conn().QueryRow("SELECT COUNT(*) FROM templates WHERE status = 'ACTIVE'").Scan(&summary.TotalTemplates)

	// Total Records across all dataset tables.
	// Kumpulkan dulu nama tabelnya dan tutup rows sebelum menghitung: pool SQLite
	// hanya punya 1 koneksi, jadi COUNT bersarang di dalam loop ini akan deadlock.
	var tables []string
	if rows, err := s.db.Conn().Query("SELECT table_name FROM datasets"); err == nil {
		for rows.Next() {
			var tbl string
			if err := rows.Scan(&tbl); err == nil {
				tables = append(tables, tbl)
			}
		}
		rows.Close()
	}

	var totalRecords int64 = 0
	for _, tbl := range tables {
		var count int64
		_ = s.db.Conn().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tbl)).Scan(&count)
		totalRecords += count
	}
	summary.TotalRecords = totalRecords

	// Total Imports
	_ = s.db.Conn().QueryRow("SELECT COUNT(*) FROM import_history").Scan(&summary.TotalImports)

	// Total Users
	_ = s.db.Conn().QueryRow("SELECT COUNT(*) FROM users").Scan(&summary.TotalUsers)

	// Database File Size
	if size, err := s.db.FileSize(); err == nil {
		summary.DatabaseSize = formatBytes(size)
	} else {
		summary.DatabaseSize = "Unknown"
	}

	// Total baris sukses / gagal sepanjang riwayat
	_ = s.db.Conn().QueryRow(
		"SELECT COALESCE(SUM(success_rows),0), COALESCE(SUM(failed_rows),0) FROM import_history",
	).Scan(&summary.SuccessRows, &summary.FailedRows)

	// Agregat untuk grafik. Semua rows ditutup sebelum query berikutnya —
	// pool SQLite hanya punya satu koneksi.
	summary.ImportTrend = s.importTrend(14)
	summary.WorkspaceSizes = s.workspaceSizes(8)
	summary.ActivityBreakdown = s.activityBreakdown(6)

	// Recent Imports
	importRows, err := s.db.Conn().Query(`
		SELECT i.id, i.template_id, t.name, i.filename, i.file_size_bytes, i.total_rows, 
		       i.success_rows, i.failed_rows, i.imported_by, i.started_at, i.finished_at, i.status, i.error_message
		FROM import_history i
		LEFT JOIN templates t ON i.template_id = t.id
		ORDER BY i.started_at DESC LIMIT 5
	`)
	if err == nil {
		defer importRows.Close()
		for importRows.Next() {
			var h models.ImportHistory
			var stStr string
			var finStr sql.NullString
			var tplName sql.NullString
			var errMsg sql.NullString

			_ = importRows.Scan(
				&h.ID, &h.TemplateID, &tplName, &h.Filename, &h.FileSizeBytes,
				&h.TotalRows, &h.SuccessRows, &h.FailedRows, &h.ImportedBy,
				&stStr, &finStr, &h.Status, &errMsg,
			)
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
			summary.RecentImports = append(summary.RecentImports, h)
		}
	}

	// Recent Activity
	summary.RecentActivity, _ = s.GetRecentLogs(6)

	return summary, nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// importTrend mengembalikan volume import per hari untuk `days` hari terakhir,
// termasuk hari tanpa aktivitas supaya sumbu waktunya tidak bolong.
func (s *ActivityService) importTrend(days int) []models.ChartPoint {
	type dayTotal struct{ ok, fail int64 }
	totals := make(map[string]dayTotal)

	rows, err := s.db.Conn().Query(`
		SELECT DATE(started_at) AS d,
		       COALESCE(SUM(success_rows),0),
		       COALESCE(SUM(failed_rows),0)
		FROM import_history
		WHERE started_at >= DATE('now', ?)
		GROUP BY d
	`, fmt.Sprintf("-%d days", days-1))
	if err == nil {
		for rows.Next() {
			var d string
			var ok, fail int64
			if err := rows.Scan(&d, &ok, &fail); err == nil {
				totals[d] = dayTotal{ok, fail}
			}
		}
		rows.Close()
	}

	out := make([]models.ChartPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		t := totals[day]
		out = append(out, models.ChartPoint{Label: day, Value: t.ok, Secondary: t.fail})
	}
	return out
}

// workspaceSizes mengembalikan jumlah baris tiap workspace, terbesar dulu
func (s *ActivityService) workspaceSizes(limit int) []models.ChartPoint {
	type ws struct {
		name  string
		table string
	}
	var list []ws

	rows, err := s.db.Conn().Query(`
		SELECT t.name, d.table_name
		FROM datasets d JOIN templates t ON t.id = d.template_id
		WHERE t.status != 'ARCHIVED'
	`)
	if err == nil {
		for rows.Next() {
			var w ws
			if err := rows.Scan(&w.name, &w.table); err == nil {
				list = append(list, w)
			}
		}
		rows.Close()
	}

	out := make([]models.ChartPoint, 0, len(list))
	for _, w := range list {
		var n int64
		_ = s.db.Conn().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", w.table)).Scan(&n)
		out = append(out, models.ChartPoint{Label: w.name, Value: n})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// activityBreakdown menghitung jumlah log per jenis aksi dalam 30 hari terakhir
func (s *ActivityService) activityBreakdown(limit int) []models.ChartPoint {
	out := make([]models.ChartPoint, 0, limit)

	rows, err := s.db.Conn().Query(`
		SELECT action, COUNT(*) AS n
		FROM activity_logs
		WHERE created_at >= DATE('now', '-30 days')
		GROUP BY action ORDER BY n DESC LIMIT ?
	`, limit)
	if err != nil {
		return out
	}
	defer rows.Close()

	for rows.Next() {
		var p models.ChartPoint
		if err := rows.Scan(&p.Label, &p.Value); err == nil {
			out = append(out, p)
		}
	}
	return out
}
