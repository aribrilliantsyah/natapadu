package settings

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"
)

type SettingsService struct {
	db *db.Database
}

func NewSettingsService(database *db.Database) *SettingsService {
	return &SettingsService{db: database}
}

// GetSetting retrieves a setting value by key
func (s *SettingsService) GetSetting(key string, defaultValue string) string {
	var val string
	err := s.db.Conn().QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return defaultValue
	}
	return val
}

// SetSetting stores a configuration key-value
func (s *SettingsService) SetSetting(key, value string) error {
	_, err := s.db.Conn().Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now(),
	)
	return err
}

// GetAllSettings returns all application configurations
func (s *SettingsService) GetAllSettings() (map[string]string, error) {
	rows, err := s.db.Conn().Query("SELECT key, value FROM app_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			settings[k] = v
		}
	}
	return settings, nil
}

// BackupDatabase copies current SQLite database to target path
func (s *SettingsService) BackupDatabase(targetDirectory string) (string, error) {
	if targetDirectory == "" {
		home, _ := os.UserHomeDir()
		targetDirectory = filepath.Join(home, "Documents", "Natapadu_Backups")
	}
	if err := os.MkdirAll(targetDirectory, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat folder backup: %w", err)
	}

	backupFileName := fmt.Sprintf("natapadu_backup_%s.db", time.Now().Format("20060102_150405"))
	destPath := filepath.Join(targetDirectory, backupFileName)

	srcFile, err := os.Open(s.db.Path())
	if err != nil {
		return "", fmt.Errorf("gagal membuka database sumber: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("gagal membuat file backup tujuan: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", fmt.Errorf("gagal menyalin file database: %w", err)
	}

	return destPath, nil
}

// SaveSavedFilter stores visual filter configuration for a template
func (s *SettingsService) SaveSavedFilter(f *models.SavedFilter) error {
	if f.ID == "" {
		f.ID = fmt.Sprintf("flt_%d", time.Now().UnixNano())
	}
	f.CreatedAt = time.Now()

	_, err := s.db.Conn().Exec(`
		INSERT INTO saved_filters (id, template_id, name, filter_payload, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		f.ID, f.TemplateID, f.Name, f.FilterPayload, f.CreatedBy, f.CreatedAt,
	)
	return err
}

// GetSavedFilters fetches saved filter presets for a template
func (s *SettingsService) GetSavedFilters(templateID string) ([]models.SavedFilter, error) {
	rows, err := s.db.Conn().Query(`
		SELECT id, template_id, name, filter_payload, created_by, created_at
		FROM saved_filters WHERE template_id = ? ORDER BY created_at DESC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.SavedFilter
	for rows.Next() {
		var f models.SavedFilter
		var cr string
		if err := rows.Scan(&f.ID, &f.TemplateID, &f.Name, &f.FilterPayload, &f.CreatedBy, &cr); err != nil {
			return nil, err
		}
		f.CreatedAt, _ = time.Parse(time.RFC3339, cr)
		list = append(list, f)
	}
	return list, nil
}

// DeleteSavedFilter deletes a saved filter preset
func (s *SettingsService) DeleteSavedFilter(filterID string) error {
	_, err := s.db.Conn().Exec("DELETE FROM saved_filters WHERE id = ?", filterID)
	return err
}
