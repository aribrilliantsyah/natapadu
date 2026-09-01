package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"natapadu-app/backend/activity"
	"natapadu-app/backend/auth"
	"natapadu-app/backend/datagrid"
	"natapadu-app/backend/db"
	"natapadu-app/backend/exporter"
	"natapadu-app/backend/importer"
	"natapadu-app/backend/models"
	"natapadu-app/backend/settings"
	"natapadu-app/backend/template"
	"natapadu-app/backend/updater"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct encapsulates desktop runtime and services
type App struct {
	ctx         context.Context
	db          *db.Database
	authSvc     *auth.AuthService
	templateSvc *template.TemplateService
	importSvc   *importer.ImportService
	dataGridSvc *datagrid.DataGridService
	exportSvc   *exporter.ExportService
	activitySvc *activity.ActivityService
	settingsSvc *settings.SettingsService
	updateSvc   *updater.Service

	version       string
	pendingUpdate string // lokasi berkas pembaruan yang sudah diunduh
}

// NewApp creates a new App application struct
func NewApp(version, updateRepo string) *App {
	database, err := db.GetDatabase()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize database: %v", err))
	}

	tplSvc := template.NewTemplateService(database)
	authSvc := auth.NewAuthService(database)
	importSvc := importer.NewImportService(database, tplSvc)
	dataGridSvc := datagrid.NewDataGridService(database, tplSvc)
	exportSvc := exporter.NewExportService(database, tplSvc)
	activitySvc := activity.NewActivityService(database)
	settingsSvc := settings.NewSettingsService(database)

	return &App{
		version:     version,
		updateSvc:   updater.New(updateRepo, version),
		db:          database,
		authSvc:     authSvc,
		templateSvc: tplSvc,
		importSvc:   importSvc,
		dataGridSvc: dataGridSvc,
		exportSvc:   exportSvc,
		activitySvc: activitySvc,
		settingsSvc: settingsSvc,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// domReady dipanggil setelah halaman selesai dimuat.
//
// Jendela HARUS ditampilkan di sini, bukan di startup. Wails menjalankan
// OnStartup pada goroutine terpisah yang berlomba dengan Window.Run(), dan
// Run() memanggil Hide() sendiri ketika StartHidden aktif. Bila WindowShow
// dipanggil dari startup, Hide() bisa menang dan jendela tidak pernah muncul
// sama sekali — persis kegagalan yang terjadi sebelumnya.
func (a *App) domReady(ctx context.Context) {
	wailsRuntime.WindowSetAlwaysOnTop(ctx, true)
	wailsRuntime.WindowCenter(ctx)
	wailsRuntime.WindowShow(ctx)
}

// ShowMainWindow membesarkan jendela dari ukuran splash ke ukuran kerja.
// Dipanggil frontend setelah persiapan awal selesai.
func (a *App) ShowMainWindow(width, height, minWidth, minHeight int) {
	wailsRuntime.WindowSetAlwaysOnTop(a.ctx, false)

	// Jendela disembunyikan dulu, diubah ukurannya, baru ditampilkan lagi.
	//
	// Di Wayland aplikasi TIDAK BOLEH memposisikan jendelanya sendiri —
	// gtk_window_move() diabaikan compositor, sehingga WindowCenter maupun
	// WindowSetPosition tidak berpengaruh sama sekali. Kalau jendela splash
	// langsung diperbesar, ia tumbuh dari sudut kiri-atasnya dan mendarat
	// melenceng sejauh setengah selisih ukuran. Menyembunyikan lalu
	// menampilkannya kembali membuat compositor menempatkannya ulang seperti
	// jendela baru — sekaligus terasa seperti splash yang ditutup lalu
	// jendela utama dibuka.
	wailsRuntime.WindowHide(a.ctx)

	// Batas minimum dinaikkan lebih dulu, lalu ukurannya — urutan terbalik
	// membuat jendela tertahan di ukuran splash pada sebagian window manager.
	wailsRuntime.WindowSetMinSize(a.ctx, minWidth, minHeight)
	wailsRuntime.WindowSetSize(a.ctx, width, height)
	wailsRuntime.WindowSetTitle(a.ctx, "Natapadu - Navigasi Master Data dan Alat Terpadu")

	// Untuk X11, Windows, dan macOS yang memang mengizinkan penempatan sendiri
	a.centerWindow(width, height)

	wailsRuntime.WindowShow(a.ctx)
}

// centerWindow memposisikan jendela berdasarkan ukuran yang DIMINTA, bukan
// ukuran yang sedang dilaporkan sistem.
//
// WindowCenter bawaan menghitung posisinya dari gtk_window_get_size(), padahal
// gtk_window_resize() barusan hanya mengajukan permintaan yang belum diterapkan
// window manager. Ukuran yang terbaca masih sebesar splash, sehingga jendela
// utama mendarat melenceng ke kanan-bawah sejauh setengah selisih ukurannya.
func (a *App) centerWindow(width, height int) {
	screens, err := wailsRuntime.ScreenGetAll(a.ctx)
	if err != nil || len(screens) == 0 {
		wailsRuntime.WindowCenter(a.ctx) // biar sistem yang menebak
		return
	}

	screen := screens[0]
	for _, s := range screens {
		if s.IsCurrent {
			screen = s
			break
		}
	}

	sw, sh := screen.Size.Width, screen.Size.Height
	if sw <= 0 || sh <= 0 {
		sw, sh = screen.Width, screen.Height // ladang lama, untuk platform yang belum mengisi Size
	}
	if sw <= 0 || sh <= 0 {
		wailsRuntime.WindowCenter(a.ctx)
		return
	}

	x, y := (sw-width)/2, (sh-height)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	wailsRuntime.WindowSetPosition(a.ctx, x, y)
}

// ==========================================
// Native Dialog & File Pickers
// ==========================================

func (a *App) SelectExcelFile() (string, error) {
	options := wailsRuntime.OpenDialogOptions{
		Title: "Pilih File Excel (.xlsx, .xlsm, .xltx)",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Excel Files (*.xlsx;*.xlsm;*.xltx)",
				Pattern:     "*.xlsx;*.xlsm;*.xltx",
			},
			{
				DisplayName: "Semua File (*.*)",
				Pattern:     "*.*",
			},
		},
	}
	return wailsRuntime.OpenFileDialog(a.ctx, options)
}

func (a *App) SelectDirectory(title string) (string, error) {
	if title == "" {
		title = "Pilih Direktori Penyimpanan"
	}
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
	})
}

// ==========================================
// Auth API
// ==========================================

func (a *App) Login(username, password string) (*models.UserSession, error) {
	sess, err := a.authSvc.Login(username, password)
	if err != nil {
		return nil, err
	}
	_ = a.activitySvc.Log(sess.User.ID, sess.User.Username, "LOGIN", "System", "Pengguna berhasil masuk")
	return sess, nil
}

func (a *App) Logout() bool {
	user := a.authSvc.GetCurrentSession()
	if user != nil {
		_ = a.activitySvc.Log(user.User.ID, user.User.Username, "LOGOUT", "System", "Pengguna keluar")
	}
	return a.authSvc.Logout()
}

func (a *App) GetCurrentUser() *models.UserSession {
	return a.authSvc.GetCurrentSession()
}

func (a *App) GetAllUsers() ([]models.User, error) {
	return a.authSvc.GetAllUsers()
}

func (a *App) CreateUser(username, password, displayName, role string) (*models.User, error) {
	user, err := a.authSvc.CreateUser(username, password, displayName, role)
	if err != nil {
		return nil, err
	}
	_ = a.activitySvc.Log("ADMIN", "System", "CREATE_USER", user.Username, fmt.Sprintf("User '%s' dibuat", user.Username))
	return user, nil
}

func (a *App) UpdatePassword(userID, newPassword string) error {
	return a.authSvc.UpdatePassword(userID, newPassword)
}

// ==========================================
// Dashboard & Activity API
// ==========================================

func (a *App) GetDashboardSummary() (*models.AppSummary, error) {
	return a.activitySvc.GetDashboardSummary()
}

func (a *App) GetRecentLogs(limit int) ([]models.ActivityLog, error) {
	return a.activitySvc.GetRecentLogs(limit)
}

// ==========================================
// Template Engine API
// ==========================================

func (a *App) GetAllTemplates() ([]models.Template, error) {
	return a.templateSvc.GetAllTemplates()
}

func (a *App) GetTemplateByID(id string) (*models.Template, error) {
	return a.templateSvc.GetTemplateByID(id)
}

func (a *App) CreateTemplate(tpl models.Template) (*models.Template, error) {
	created, err := a.templateSvc.CreateTemplate(&tpl)
	if err != nil {
		return nil, err
	}
	_ = a.activitySvc.Log("", "", "CREATE_TEMPLATE", created.Name, fmt.Sprintf("Template '%s' berhasil dibuat", created.Name))
	return created, nil
}

func (a *App) UpdateTemplate(tpl models.Template) (*models.Template, error) {
	updated, err := a.templateSvc.UpdateTemplate(&tpl)
	if err != nil {
		return nil, err
	}
	_ = a.activitySvc.Log("", "", "UPDATE_TEMPLATE", updated.Name, fmt.Sprintf("Template '%s' diperbarui", updated.Name))
	return updated, nil
}

func (a *App) DeleteTemplate(id string) error {
	t, _ := a.templateSvc.GetTemplateByID(id)
	name := id
	if t != nil {
		name = t.Name
	}
	err := a.templateSvc.DeleteTemplate(id)
	if err != nil {
		return err
	}
	_ = a.activitySvc.Log("", "", "DELETE_TEMPLATE", name, fmt.Sprintf("Template '%s' dihapus", name))
	return nil
}

func (a *App) DuplicateTemplate(sourceID string, newName string) (*models.Template, error) {
	cloned, err := a.templateSvc.DuplicateTemplate(sourceID, newName)
	if err != nil {
		return nil, err
	}
	_ = a.activitySvc.Log("", "", "DUPLICATE_TEMPLATE", cloned.Name, fmt.Sprintf("Template '%s' berhasil diduplikasi", cloned.Name))
	return cloned, nil
}

func (a *App) ExportTemplateSchema(templateID string) (string, error) {
	return a.templateSvc.ExportTemplateSchema(templateID)
}

func (a *App) ImportTemplateSchema(jsonStr string) (*models.Template, error) {
	return a.templateSvc.ImportTemplateSchema(jsonStr)
}

// ==========================================
// Importer API
// ==========================================

func (a *App) PreviewExcelFile(filePath, sheetName string, headerRow, sampleLimit int) (*models.ExcelSheetPreview, error) {
	return a.importSvc.PreviewExcelFile(filePath, sheetName, headerRow, sampleLimit)
}

func (a *App) StartImport(templateID, filePath, importedBy string) (*models.ImportHistory, error) {
	if importedBy == "" {
		if sess := a.authSvc.GetCurrentSession(); sess != nil {
			importedBy = sess.User.Username
		} else {
			importedBy = "Operator"
		}
	}

	progressHandler := func(ev models.ImportProgressEvent) {
		wailsRuntime.EventsEmit(a.ctx, "import:progress", ev)
	}

	history, err := a.importSvc.ExecuteImport(a.ctx, templateID, filePath, importedBy, progressHandler)
	if err != nil {
		_ = a.activitySvc.Log("", importedBy, "IMPORT_FAILED", filePath, err.Error())
		return nil, err
	}

	_ = a.activitySvc.Log(
		"", importedBy, "IMPORT_SUCCESS", history.Filename,
		fmt.Sprintf("Import selesai: %d sukses, %d gagal", history.SuccessRows, history.FailedRows),
	)
	return history, nil
}

func (a *App) CancelImport(importID string) bool {
	return a.importSvc.CancelImport(importID)
}

func (a *App) GetImportHistory(templateID string, limit int) ([]models.ImportHistory, error) {
	return a.importSvc.GetImportHistory(templateID, limit)
}

func (a *App) GetImportErrors(importID string, limit int) ([]models.ImportError, error) {
	return a.importSvc.GetImportErrors(importID, limit)
}

// ==========================================
// Data Viewer / Grid API
// ==========================================

func (a *App) QueryData(req models.QueryRequest) (*models.QueryResponse, error) {
	return a.dataGridSvc.QueryData(req)
}

func (a *App) DeleteRow(templateID string, rowID int64) error {
	return a.dataGridSvc.DeleteRow(templateID, rowID)
}

func (a *App) BulkDeleteRows(templateID string, rowIDs []int64) (int64, error) {
	return a.dataGridSvc.BulkDeleteRows(templateID, rowIDs)
}

func (a *App) TruncateDataset(templateID string) error {
	t, _ := a.templateSvc.GetTemplateByID(templateID)
	name := templateID
	if t != nil {
		name = t.Name
	}
	err := a.dataGridSvc.TruncateDataset(templateID)
	if err == nil {
		_ = a.activitySvc.Log("", "", "TRUNCATE_DATASET", name, fmt.Sprintf("Seluruh data dataset '%s' dikosongkan", name))
	}
	return err
}

// SaveDataRow menyimpan satu baris manual (rowID <= 0 berarti baris baru).
// Nilai mentah dilewatkan ke pipeline transform+validasi yang sama dengan import,
// supaya data hasil ketik tangan tidak lolos aturan yang berlaku untuk data hasil import.
func (a *App) SaveDataRow(templateID string, rowID int64, raw map[string]string) (int64, error) {
	tpl, err := a.templateSvc.GetTemplateByID(templateID)
	if err != nil {
		return 0, err
	}

	values := make(map[string]interface{}, len(tpl.Columns))
	for _, c := range tpl.Columns {
		v, err := a.importSvc.TransformAndValidate(raw[c.FieldName], c)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", c.DisplayName, err)
		}
		values[c.FieldName] = v
	}

	savedID, err := a.dataGridSvc.UpsertRow(templateID, rowID, values)
	if err != nil {
		return 0, err
	}

	action := "CREATE_ROW"
	if rowID > 0 {
		action = "UPDATE_ROW"
	}
	_ = a.activitySvc.Log("", "", action, tpl.Name, fmt.Sprintf("Baris #%d disimpan manual di workspace '%s'", savedID, tpl.Name))
	return savedID, nil
}

func (a *App) GetDataRow(templateID string, rowID int64) (map[string]interface{}, error) {
	return a.dataGridSvc.GetRow(templateID, rowID)
}

// ==========================================
// Exporter API
// ==========================================

// ExportData menulis dataset ke XLSX, CSV, atau ODS.
// Mengembalikan satu struct: binding Wails hanya mendukung (value, error), sehingga
// signature lama (string, int64, error) menghasilkan null di sisi frontend.
func (a *App) ExportData(req models.ExportRequest, saveDirectory string) (*models.ExportResult, error) {
	res, err := a.exportSvc.Export(req, saveDirectory)
	if err != nil {
		return nil, err
	}
	_ = a.activitySvc.Log("", "", "EXPORT", res.FilePath, fmt.Sprintf("Export %d baris (%s) ke %s", res.RowCount, res.Format, res.FilePath))
	return res, nil
}

// GetDistinctValues melayani penjelajahan data per kelompok nilai kolom
func (a *App) GetDistinctValues(templateID, fieldName, search string, limit int) ([]models.DistinctValue, error) {
	return a.dataGridSvc.GetDistinctValues(templateID, fieldName, search, limit)
}

// GetDuplicateGroups mencari kombinasi nilai yang berulang pada satu atau beberapa kolom
func (a *App) GetDuplicateGroups(templateID string, fields []string, search string, limit int) ([]models.DuplicateGroup, error) {
	return a.dataGridSvc.GetDuplicateGroups(templateID, fields, search, limit)
}

// DownloadDataTemplate menghasilkan file Excel kosong (hanya header, sesuai layout workspace)
// untuk diisi pengguna lalu di-import kembali ke workspace yang sama.
func (a *App) DownloadDataTemplate(templateID, saveDirectory string) (string, error) {
	filePath, err := a.exportSvc.ExportBlankTemplate(templateID, saveDirectory)
	if err != nil {
		return "", err
	}
	_ = a.activitySvc.Log("", "", "DOWNLOAD_TEMPLATE", templateID, fmt.Sprintf("Template pengisian dibuat di %s", filePath))
	return filePath, nil
}

// ==========================================
// Update API
// ==========================================

// GetAppVersion mengembalikan versi yang sedang berjalan ("dev" untuk build lokal)
func (a *App) GetAppVersion() string {
	return a.version
}

// CheckForUpdate menanyakan rilis terbaru ke GitHub.
// Hanya memeriksa — tidak pernah mengunduh atau memasang tanpa perintah pengguna.
func (a *App) CheckForUpdate() (*updater.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	info, err := a.updateSvc.Check(ctx)
	if err != nil {
		return nil, err
	}
	if info.Available {
		_ = a.activitySvc.Log("", "", "UPDATE_AVAILABLE", info.LatestVersion,
			fmt.Sprintf("Versi %s tersedia (sedang berjalan %s)", info.LatestVersion, info.CurrentVersion))
	}
	return info, nil
}

// DownloadUpdate mengunduh berkas rilis, mengirim kemajuannya lewat event
// "update:progress" agar UI bisa menampilkan bilah kemajuan.
func (a *App) DownloadUpdate(downloadURL, assetName string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	path, err := a.updateSvc.Download(ctx, downloadURL, assetName, func(downloaded, total int64) {
		wailsRuntime.EventsEmit(a.ctx, "update:progress", map[string]int64{
			"downloaded": downloaded,
			"total":      total,
		})
	})
	if err != nil {
		return "", err
	}

	a.pendingUpdate = path
	_ = a.activitySvc.Log("", "", "UPDATE_DOWNLOADED", assetName, fmt.Sprintf("Berkas pembaruan diunduh ke %s", path))
	return path, nil
}

// InstallUpdate memasang berkas yang sudah diunduh.
// Di Linux (AppImage) berkas lama diganti dan disimpan sebagai cadangan;
// di Windows berkas diserahkan ke pengguna karena .exe yang sedang berjalan
// tidak bisa menimpa dirinya sendiri.
func (a *App) InstallUpdate() (*updater.InstallResult, error) {
	if a.pendingUpdate == "" {
		return nil, errors.New("belum ada berkas pembaruan yang diunduh")
	}

	res, err := updater.Install(a.pendingUpdate)
	if err != nil {
		return nil, err
	}
	if res.Installed {
		_ = a.activitySvc.Log("", "", "UPDATE_INSTALLED", res.Path, "Pembaruan terpasang, menunggu aplikasi dijalankan ulang")
	}
	return res, nil
}

// OpenUpdateFolder membuka lokasi berkas pembaruan di pengelola berkas sistem
func (a *App) OpenUpdateFolder() error {
	if a.pendingUpdate == "" {
		return errors.New("belum ada berkas pembaruan yang diunduh")
	}
	wailsRuntime.BrowserOpenURL(a.ctx, "file://"+filepath.Dir(a.pendingUpdate))
	return nil
}

// OpenReleasePage membuka halaman rilis di peramban
func (a *App) OpenReleasePage(url string) {
	if url != "" {
		wailsRuntime.BrowserOpenURL(a.ctx, url)
	}
}

// ==========================================
// Settings, Saved Filters & Backup API
// ==========================================

func (a *App) GetSetting(key, defaultValue string) string {
	return a.settingsSvc.GetSetting(key, defaultValue)
}

func (a *App) SetSetting(key, value string) error {
	return a.settingsSvc.SetSetting(key, value)
}

func (a *App) GetAllSettings() (map[string]string, error) {
	return a.settingsSvc.GetAllSettings()
}

func (a *App) BackupDatabase(targetDirectory string) (string, error) {
	path, err := a.settingsSvc.BackupDatabase(targetDirectory)
	if err != nil {
		return "", err
	}
	_ = a.activitySvc.Log("", "", "BACKUP", "Database", fmt.Sprintf("Backup dibuat di %s", path))
	return path, nil
}

func (a *App) SaveSavedFilter(f models.SavedFilter) error {
	return a.settingsSvc.SaveSavedFilter(&f)
}

func (a *App) GetSavedFilters(templateID string) ([]models.SavedFilter, error) {
	return a.settingsSvc.GetSavedFilters(templateID)
}

func (a *App) DeleteSavedFilter(filterID string) error {
	return a.settingsSvc.DeleteSavedFilter(filterID)
}
