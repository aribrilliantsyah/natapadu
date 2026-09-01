package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// Version diisi saat build lewat -ldflags "-X main.Version=1.2.3".
// Nilai "dev" berarti build lokal, dan selalu dianggap lebih tua daripada
// rilis mana pun sehingga pengembang tetap melihat tawaran pembaruan.
var Version = "dev"

// UpdateRepo adalah repositori GitHub tempat rilis dicari.
const UpdateRepo = "aribrilliantsyah/natapadu"

// Ukuran jendela. Aplikasi dibuka sebagai kotak splash kecil di tengah layar,
// lalu dibesarkan ke ukuran kerja setelah persiapan selesai.
const (
	splashWidth  = 460
	splashHeight = 280

	mainWidth     = 1366
	mainHeight    = 860
	mainMinWidth  = 1024
	mainMinHeight = 700
)

func main() {
	// Create an instance of the app structure
	app := NewApp(Version, UpdateRepo)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Natapadu",
		Width:  splashWidth,
		Height: splashHeight,
		// Batas minimum menyusul setelah splash; kalau dipasang sekarang,
		// jendela tidak bisa mengecil ke ukuran splash.
		MinWidth:  splashWidth,
		MinHeight: splashHeight,
		// Jendela ditahan sampai splash siap digambar, supaya tidak ada
		// kedipan kotak kosong sebelum isinya muncul.
		StartHidden: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 255}, // slate-900
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
