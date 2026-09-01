package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Perbandingan versi adalah penentu apakah pengguna ditawari pembaruan.
// Salah di sini berarti pembaruan tidak pernah muncul, atau muncul terus-menerus.
func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.0.0", "1.0.0", 0},
		{"v1.2.0", "1.2.0", 0}, // awalan v tidak berpengaruh
		{"1.10.0", "1.9.0", 1}, // dibandingkan sebagai angka, bukan teks
		{"2.0.0", "1.99.99", 1},
		{"1.2", "1.2.0", 0}, // bagian yang hilang dianggap nol
		{"1.2.1", "1.2", 1},
		{"1.0.0", "dev", 1}, // build pengembangan selalu dianggap lebih tua
		{"dev", "1.0.0", -1},
		{"dev", "dev", 0},
		{"1.0.0", "", 1},             // versi kosong diperlakukan sebagai dev
		{"1.2.0", "1.2.0-beta.1", 1}, // rilis final mengalahkan pra-rilis
		{"1.2.0-beta.1", "1.2.0", -1},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, mau %d", c.a, c.b, got, c.want)
		}
	}
}

func releaseServer(t *testing.T, rel ghRelease) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	}))
}

func checkAgainst(t *testing.T, srv *httptest.Server, current string) (*Info, error) {
	t.Helper()
	s := New("uji/repo", current)
	// Arahkan ke server uji tanpa mengubah kode produksi
	old := releasesAPIFor
	releasesAPIFor = func(string) string { return srv.URL }
	t.Cleanup(func() { releasesAPIFor = old })
	return s.Check(context.Background())
}

func TestCheckFindsNewerRelease(t *testing.T) {
	asset := ghAsset{Name: "Natapadu-x86_64.AppImage", Size: 1234, BrowserDownloadURL: "https://contoh/app"}
	srv := releaseServer(t, ghRelease{
		TagName: "v2.0.0", Body: "Perbaikan bug", HTMLURL: "https://contoh/rilis",
		Assets: []ghAsset{{Name: "sumber.zip"}, asset, {Name: "Natapadu-windows-amd64.exe"}},
	})
	defer srv.Close()

	info, err := checkAgainst(t, srv, "1.0.0")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !info.Available {
		t.Fatal("pembaruan seharusnya tersedia")
	}
	if info.LatestVersion != "2.0.0" {
		t.Errorf("versi = %q, mau 2.0.0 (awalan v harus dilepas)", info.LatestVersion)
	}
	if info.ReleaseNotes != "Perbaikan bug" {
		t.Errorf("catatan rilis tidak terbawa")
	}
}

func TestCheckSameVersionOffersNothing(t *testing.T) {
	srv := releaseServer(t, ghRelease{TagName: "v1.0.0"})
	defer srv.Close()

	info, err := checkAgainst(t, srv, "1.0.0")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if info.Available {
		t.Error("versi sama tidak boleh ditawari pembaruan")
	}
}

// Versi berjalan lebih baru dari rilis (mis. build lokal) juga tidak boleh
// menawarkan "pembaruan" yang justru menurunkan versi.
func TestCheckOlderReleaseOffersNothing(t *testing.T) {
	srv := releaseServer(t, ghRelease{TagName: "v1.0.0"})
	defer srv.Close()

	info, err := checkAgainst(t, srv, "1.5.0")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if info.Available {
		t.Error("rilis lebih lama tidak boleh ditawarkan")
	}
}

func TestCheckDraftIgnored(t *testing.T) {
	srv := releaseServer(t, ghRelease{TagName: "v9.0.0", Draft: true})
	defer srv.Close()

	info, err := checkAgainst(t, srv, "1.0.0")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if info.Available {
		t.Error("rilis draft tidak boleh ditawarkan")
	}
}

// Rilis ada tapi tanpa berkas untuk platform ini: harus jujur, bukan
// menawarkan tombol pasang yang tidak mengunduh apa pun.
func TestCheckWithoutMatchingAsset(t *testing.T) {
	srv := releaseServer(t, ghRelease{
		TagName: "v3.0.0",
		Assets:  []ghAsset{{Name: "catatan.txt"}},
	})
	defer srv.Close()

	info, err := checkAgainst(t, srv, "1.0.0")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !info.Available {
		t.Fatal("pembaruan tetap harus terdeteksi")
	}
	if info.DownloadURL != "" {
		t.Error("tidak boleh ada tautan unduhan bila berkasnya tidak ada")
	}
	if info.Note == "" {
		t.Error("harus menjelaskan kenapa tidak bisa diunduh")
	}
}

func TestDownloadWritesCompleteFile(t *testing.T) {
	payload := strings.Repeat("natapadu", 5000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	s := New("uji/repo", "1.0.0")
	var lastDownloaded int64
	path, err := s.Download(context.Background(), srv.URL, "Natapadu.AppImage", func(d, total int64) {
		lastDownloaded = d
	})
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	if filepath.Base(path) != "Natapadu.AppImage" {
		t.Errorf("nama berkas = %s", filepath.Base(path))
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("baca hasil: %v", err)
	}
	if string(got) != payload {
		t.Errorf("isi berkas tidak utuh: %d dari %d byte", len(got), len(payload))
	}
	if lastDownloaded != int64(len(payload)) {
		t.Errorf("laporan kemajuan terakhir = %d, mau %d", lastDownloaded, len(payload))
	}
	// Berkas sementara tidak boleh tertinggal
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Error("berkas .part tidak dibersihkan")
	}
}

func TestDownloadRejectsErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "hilang", http.StatusNotFound)
	}))
	defer srv.Close()

	s := New("uji/repo", "1.0.0")
	if _, err := s.Download(context.Background(), srv.URL, "x.AppImage", nil); err == nil {
		t.Error("unduhan gagal harus mengembalikan error, bukan berkas kosong")
	}
}
