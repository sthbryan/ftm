package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sthbryan/ftm/internal/app"
	"github.com/sthbryan/ftm/internal/autostart"
	"github.com/sthbryan/ftm/internal/buildinfo"
	"github.com/sthbryan/ftm/internal/platform/instance"
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

	// When ftm already runs (the TUI, ftm --server or another window), this
	// window shows that process's dashboard instead of starting a second
	// manager. ftmApp stays nil then.
	var webURL string
	ftmApp, err := app.New()
	var running *instance.RunningError
	switch {
	case errors.As(err, &running):
		if running.Info == nil {
			fmt.Fprintln(os.Stderr, running.Error())
			os.Exit(1)
		}
		webURL = running.Info.URL()
		fmt.Printf("Attaching to the running instance at %s\n", webURL)
	case err != nil:
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	default:
		if port > 0 {
			if err := ftmApp.SetWebPort(port); err != nil {
				log.Printf("Could not save the web port: %v", err)
			}
		}

		if err := ftmApp.StartWebServer(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
			ftmApp.Shutdown()
			os.Exit(1)
		}

		webURL = ftmApp.WebServer.URL()
		fmt.Printf("Foundry Tunnel Manager v%s\n", version.Version)
		fmt.Printf("Desktop app running at: %s\n", webURL)
	}

	wailsApp := application.New(application.Options{
		Name:        "Foundry Tunnel Manager",
		Description: "Share your Foundry VTT world without port forwarding",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			log.Println("App shutting down...")
			if ftmApp != nil {
				ftmApp.Shutdown()
			}
		},
	})

	if ftmApp != nil {
		ftmApp.WebServer.SetPiPOpener(func(tunnelID string) error {
			application.InvokeAsync(func() {
				openPiPWindow(wailsApp, webURL, tunnelID)
			})
			return nil
		})
	}

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
