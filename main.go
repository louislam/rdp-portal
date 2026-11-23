package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// The app is portable: config.json and rdp-files/ live next to the exe.
	if exe, err := os.Executable(); err == nil {
		_ = os.Chdir(dirOf(exe))
	}

	svc, err := NewPortalService()
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "RDP Portal",
		Description: "A slightly smarter and simple Remote Desktop manager for Windows.",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			// Set RDP_PORTAL_CDP_PORT to expose WebView2 devtools protocol,
			// which the end-to-end tests drive.
			AdditionalBrowserArgs: cdpArgs(),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.louislam.rdpportal",
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "RDP Portal",
		Width:            700,
		Height:           620,
		MinWidth:         640,
		MinHeight:        560,
		URL:              "/",
		BackgroundColour: application.NewRGB(240, 240, 240),
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == os.PathSeparator || path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

// cdpArgs exposes the WebView2 devtools protocol when RDP_PORTAL_CDP_PORT is set.
func cdpArgs() []string {
	if port := os.Getenv("RDP_PORTAL_CDP_PORT"); port != "" {
		return []string{"--remote-debugging-port=" + port}
	}
	return nil
}



