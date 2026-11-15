package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sthbryan/ftm/internal/app"
	"github.com/sthbryan/ftm/internal/autostart"
	"github.com/sthbryan/ftm/internal/buildinfo"
	"github.com/sthbryan/ftm/internal/version"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 0, "Web server port")
	flag.Parse()

	buildinfo.MarkDesktop()

	if err := autostart.New().Repair(); err != nil {
		log.Printf("Could not refresh the autostart entry: %v", err)
	}

	ftmApp, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if port > 0 {
		ftmApp.Config.WebPort = port
	}

	if err := ftmApp.StartWebServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
		os.Exit(1)
	}

	webURL := ftmApp.WebServer.URL()
	fmt.Printf("Foundry Tunnel Manager v%s\n", version.Version)
	fmt.Printf("Desktop app running at: %s\n", webURL)

	wailsApp := application.New(application.Options{
		Name:        "Foundry Tunnel Manager",
		Description: "Share your Foundry VTT world with your players",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			log.Println("App shutting down...")
			ftmApp.Shutdown()
		},
	})

	ftmApp.WebServer.SetPiPOpener(func(tunnelID string) error {
		application.InvokeAsync(func() {
			openPiPWindow(wailsApp, webURL, tunnelID)
		})
		return nil
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Foundry Tunnel Manager",
		Width:            1200,
		Height:           800,
		MinWidth:         900,
		MinHeight:        620,
		URL:              webURL,
		BackgroundColour: application.NewRGB(255, 255, 255),
		DevToolsEnabled:  false,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})

	if err := wailsApp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Wails: %v\n", err)
		os.Exit(1)
	}
}
