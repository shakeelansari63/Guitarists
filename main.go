package main

import (
	"embed"
	"log"
	"time"

	"github.com/freak/guitarists/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	tunerService := services.NewTunerService()
	metronomeService := services.NewMetronomeService()
	systemService := services.NewSystemService(version)

	tabService, err := services.NewTabService()
	if err != nil {
		log.Fatalf("initialising tabs: %v", err)
	}

	app := application.New(application.Options{
		Name:        "Guitarists",
		Description: "Tools for expert guitarists: tuner, metronome, and song editor.",
		Icon:        appIcon,
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Services: []application.Service{
			application.NewService(tunerService),
			application.NewService(metronomeService),
			application.NewService(tabService),
			application.NewService(systemService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Guitarists",
		Width:            1440,
		Height:           900,
		MinWidth:         900,
		MinHeight:        620,
		InitialPosition:  application.WindowCentered,
		Hidden:           true,
		BackgroundColour: application.NewRGB(18, 20, 24),
		URL:              "/",
	})

	time.AfterFunc(2*time.Second, func() {
		win.Show()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
