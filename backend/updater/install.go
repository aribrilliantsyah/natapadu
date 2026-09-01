package updater

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// InstallResult memberitahu UI apa yang sudah terjadi dan apa yang perlu
// dilakukan pengguna berikutnya.
type InstallResult struct {
	Installed    bool   `json:"installed"`    // berkas program sudah diganti
	NeedsRestart bool   `json:"needsRestart"` // cukup tutup-buka aplikasi
	Path         string `json:"path"`         // lokasi berkas hasil unduhan
	Message      string `json:"message"`
}

// Install memasang berkas yang sudah diunduh.
//
// Linux (AppImage) dipasang otomatis: berkas lama diganti, yang lama disimpan
// sebagai .old sebagai jalan mundur bila yang baru bermasalah.
//
// Windows tidak dipasang otomatis — berkas .exe yang sedang berjalan tidak bisa
// ditimpa oleh dirinya sendiri, dan menyiasatinya butuh proses pembantu yang
// berjalan setelah aplikasi tertutup. Menerapkannya setengah jadi berisiko
// meninggalkan pemasangan yang rusak, jadi berkasnya diserahkan ke pengguna.
func Install(downloadedPath string) (*InstallResult, error) {
	if downloadedPath == "" {
		return nil, fmt.Errorf("tidak ada berkas pembaruan")
	}
	if _, err := os.Stat(downloadedPath); err != nil {
		return nil, fmt.Errorf("berkas pembaruan tidak ditemukan: %w", err)
	}

	if runtime.GOOS == "linux" {
		if appImage := os.Getenv("APPIMAGE"); appImage != "" {
			return installAppImage(downloadedPath, appImage)
		}
	}

	return &InstallResult{
		Installed: false,
		Path:      downloadedPath,
		Message:   manualInstructions(downloadedPath),
	}, nil
}

// installAppImage mengganti berkas AppImage yang sedang berjalan.
// Aman dilakukan di Linux: berkas yang sedang dieksekusi boleh diganti nama,
// proses yang berjalan tetap memakai inode lama sampai ditutup.
func installAppImage(downloaded, target string) (*InstallResult, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("AppImage aktif tidak terbaca: %w", err)
	}

	// Pastikan lokasinya bisa ditulis sebelum menyentuh apa pun
	if err := checkWritable(filepath.Dir(target)); err != nil {
		return &InstallResult{
			Installed: false,
			Path:      downloaded,
			Message: fmt.Sprintf(
				"Tidak punya izin menulis di %s. Salin berkas berikut ke sana secara manual:\n%s",
				filepath.Dir(target), downloaded,
			),
		}, nil
	}

	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		return nil, fmt.Errorf("gagal menyisihkan versi lama: %w", err)
	}

	if err := copyFile(downloaded, target, info.Mode()); err != nil {
		// Kembalikan versi lama agar aplikasi tetap bisa dijalankan
		_ = os.Rename(backup, target)
		return nil, fmt.Errorf("gagal memasang versi baru, versi lama dikembalikan: %w", err)
	}
	if err := os.Chmod(target, 0755); err != nil {
		return nil, fmt.Errorf("gagal memberi izin eksekusi: %w", err)
	}

	return &InstallResult{
		Installed:    true,
		NeedsRestart: true,
		Path:         target,
		Message: fmt.Sprintf(
			"Pembaruan terpasang. Tutup lalu buka kembali aplikasi untuk memakai versi baru.\nVersi lama disimpan di %s.",
			backup,
		),
	}, nil
}

func manualInstructions(path string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf(
			"Berkas pembaruan sudah diunduh ke:\n%s\n\n"+
				"Tutup aplikasi ini, lalu ganti berkas .exe lama dengan yang baru dan jalankan kembali.",
			path)
	default:
		return fmt.Sprintf(
			"Berkas pembaruan sudah diunduh ke:\n%s\n\n"+
				"Ganti berkas program lama dengan berkas ini, lalu jalankan kembali.",
			path)
	}
}

func checkWritable(dir string) error {
	probe, err := os.CreateTemp(dir, ".natapadu-write-test-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	probe.Close()
	return os.Remove(name)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
