package main

import (
	"embed"
	"log"

	"github.com/freak/guitarists/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const version = "0.1.0"

//go:embed all:frontend/dist
var assets embed.FS

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
		Description: "Tools for expert guitarists: tuner, metronome, and tab editor.",
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

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Guitarists",
		Width:            1180,
		Height:           800,
		MinWidth:         900,
		MinHeight:        620,
		BackgroundColour: application.NewRGB(18, 20, 24),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
