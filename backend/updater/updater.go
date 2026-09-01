// Package updater memeriksa rilis baru di GitHub dan mengunduhnya.
//
// Perilaku yang disengaja: aplikasi hanya MEMERIKSA secara otomatis, tidak
// pernah mengunduh atau memasang tanpa perintah pengguna. Mengganti berkas
// program milik orang lain diam-diam bukan sesuatu yang boleh terjadi begitu saja.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	releasesAPI = "https://api.github.com/repos/%s/releases/latest"
	httpTimeout = 20 * time.Second
)

// Service memeriksa dan mengunduh pembaruan aplikasi.
// releasesAPIFor dibungkus variabel supaya pengujian bisa mengarahkannya
// ke server tiruan tanpa mengubah alur produksi.
var releasesAPIFor = func(repo string) string {
	return fmt.Sprintf(releasesAPI, repo)
}

type Service struct {
	repo    string // "pemilik/repo"
	current string // versi yang sedang berjalan
	client  *http.Client
}

func New(repo, currentVersion string) *Service {
	return &Service{
		repo:    repo,
		current: currentVersion,
		client:  &http.Client{Timeout: httpTimeout},
	}
}

// Info adalah hasil pemeriksaan pembaruan yang dikirim ke UI.
type Info struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseNotes   string `json:"releaseNotes"`
	ReleaseURL     string `json:"releaseUrl"`
	DownloadURL    string `json:"downloadUrl"`
	AssetName      string `json:"assetName"`
	AssetSize      int64  `json:"assetSize"`
	PublishedAt    string `json:"publishedAt"`
	// Terisi bila rilisnya ada tapi tidak menyediakan berkas untuk platform ini
	Note string `json:"note"`
}

type ghAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

// Check menanyakan rilis terbaru ke GitHub dan membandingkannya dengan versi berjalan.
func (s *Service) Check(ctx context.Context) (*Info, error) {
	info := &Info{CurrentVersion: s.current}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPIFor(s.repo), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "natapadu-updater")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tidak bisa menghubungi GitHub: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, fmt.Errorf("belum ada rilis yang diterbitkan di %s", s.repo)
	case http.StatusForbidden:
		return nil, fmt.Errorf("batas permintaan GitHub tercapai, coba lagi nanti")
	default:
		return nil, fmt.Errorf("GitHub menjawab %s", resp.Status)
	}

	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("jawaban GitHub tidak terbaca: %w", err)
	}
	if rel.Draft {
		return info, nil
	}

	info.LatestVersion = strings.TrimPrefix(rel.TagName, "v")
	info.ReleaseNotes = rel.Body
	info.ReleaseURL = rel.HTMLURL
	if !rel.PublishedAt.IsZero() {
		info.PublishedAt = rel.PublishedAt.Format("2006-01-02")
	}

	if CompareVersions(info.LatestVersion, s.current) <= 0 {
		return info, nil // sudah versi terbaru
	}
	info.Available = true

	if asset := pickAsset(rel.Assets); asset != nil {
		info.DownloadURL = asset.BrowserDownloadURL
		info.AssetName = asset.Name
		info.AssetSize = asset.Size
	} else {
		info.Note = fmt.Sprintf("Rilis %s tersedia, tapi belum menyediakan berkas untuk %s.", info.LatestVersion, runtime.GOOS)
	}
	return info, nil
}

// pickAsset memilih berkas rilis yang sesuai sistem operasi yang sedang berjalan.
func pickAsset(assets []ghAsset) *ghAsset {
	var wantSuffix []string
	switch runtime.GOOS {
	case "linux":
		wantSuffix = []string{".appimage"}
	case "windows":
		wantSuffix = []string{".exe"}
	case "darwin":
		wantSuffix = []string{".dmg", ".zip"}
	default:
		return nil
	}

	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		for _, suf := range wantSuffix {
			if strings.HasSuffix(name, suf) {
				return &assets[i]
			}
		}
	}
	return nil
}

var versionPart = regexp.MustCompile(`\d+`)

// CompareVersions membandingkan dua versi bergaya semver.
// Mengembalikan 1 bila a lebih baru, -1 bila b lebih baru, 0 bila setara.
// Versi pembangunan ("dev") selalu dianggap lebih tua agar pengembang tetap
// melihat tawaran pembaruan.
func CompareVersions(a, b string) int {
	a, b = normalizeVersion(a), normalizeVersion(b)
	if a == b {
		return 0
	}
	if b == "dev" {
		return 1
	}
	if a == "dev" {
		return -1
	}

	// Pisahkan bagian pra-rilis lebih dulu. Tanpa ini, angka di "-beta.1" ikut
	// terbaca sebagai komponen versi keempat sehingga 1.2.0-beta.1 keliru
	// dianggap lebih baru daripada 1.2.0.
	coreA, preRelA := splitPreRelease(a)
	coreB, preRelB := splitPreRelease(b)

	pa, pb := versionPart.FindAllString(coreA, -1), versionPart.FindAllString(coreB, -1)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		na, nb := 0, 0
		if i < len(pa) {
			na, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			nb, _ = strconv.Atoi(pb[i])
		}
		if na != nb {
			if na > nb {
				return 1
			}
			return -1
		}
	}

	// Angka inti sama: rilis final mengalahkan pra-rilis (1.2.0 > 1.2.0-beta.1)
	switch {
	case preRelA != "" && preRelB == "":
		return -1
	case preRelA == "" && preRelB != "":
		return 1
	case preRelA > preRelB:
		return 1
	case preRelA < preRelB:
		return -1
	}
	return 0
}

// splitPreRelease memisahkan "1.2.0-beta.1" menjadi ("1.2.0", "beta.1").
func splitPreRelease(v string) (core, pre string) {
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return "dev"
	}
	return v
}

// ProgressFunc melaporkan kemajuan unduhan ke UI.
type ProgressFunc func(downloaded, total int64)

// Download mengunduh berkas rilis ke direktori sementara dan mengembalikan lokasinya.
// Berkas ditulis dengan nama sementara lalu diganti nama setelah utuh, sehingga
// unduhan yang terputus tidak pernah menghasilkan berkas separuh yang tampak sah.
func (s *Service) Download(ctx context.Context, url, filename string, onProgress ProgressFunc) (string, error) {
	if url == "" {
		return "", fmt.Errorf("tidak ada berkas untuk diunduh")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "natapadu-updater")

	// Unduhan berkas besar tidak boleh terikat batas waktu pemeriksaan
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal mengunduh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unduhan ditolak: %s", resp.Status)
	}

	dir := filepath.Join(os.TempDir(), "natapadu-update")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if filename == "" {
		filename = "natapadu-update"
	}

	finalPath := filepath.Join(dir, filepath.Base(filename))
	tmpPath := finalPath + ".part"

	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 256*1024)
	lastReport := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				out.Close()
				os.Remove(tmpPath)
				return "", err
			}
			written += int64(n)
			// Dibatasi agar tidak membanjiri event bus pada koneksi cepat
			if onProgress != nil && time.Since(lastReport) > 150*time.Millisecond {
				onProgress(written, total)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("unduhan terputus: %w", readErr)
		}
		if ctx.Err() != nil {
			out.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("unduhan dibatalkan")
		}
	}

	if err := out.Sync(); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}

	// Berkas tidak utuh lebih berbahaya daripada gagal terang-terangan
	if total > 0 && written != total {
		os.Remove(tmpPath)
		return "", fmt.Errorf("unduhan tidak lengkap: %d dari %d byte", written, total)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", err
	}
	if onProgress != nil {
		onProgress(written, written)
	}
	return finalPath, nil
}
