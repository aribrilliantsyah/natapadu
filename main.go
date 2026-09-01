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

func main() {
	// Create an instance of the app structure
	app := NewApp(Version, UpdateRepo)

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Natapadu - Navigasi Master Data dan Alat Terpadu",
		Width:     1366,
		Height:    860,
		MinWidth:  1024,
		MinHeight: 700,
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
