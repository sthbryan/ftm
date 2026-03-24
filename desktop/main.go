package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sthbryan/ftm/internal/app"
	"github.com/sthbryan/ftm/internal/version"
	"github.com/wailsapp/wails/v2"
	wailsassetserver "github.com/wailsapp/wails/v2/pkg/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	var port int
	flag.IntVar(&port, "port", 0, "Web server port")
	flag.Parse()

	application, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if port > 0 {
		application.Config.WebPort = port
	}

	if err := application.StartWebServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
		os.Exit(1)
	}

	webURL := application.WebServer.URL()
	fmt.Printf("Foundry Tunnel Manager v%s\n", version.Version)
	fmt.Printf("Desktop app running at: %s\n", webURL)

	proxyHandler := wailsassetserver.NewProxyServer(webURL)

	err = wails.Run(&options.App{
		Title:  "Foundry Tunnel Manager",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets:  nil,
			Handler: proxyHandler,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
		},
		OnStartup: func(ctx context.Context) {
			log.Println("App starting...")
		},
		OnShutdown: func(ctx context.Context) {
			log.Println("App shutting down...")
			application.Shutdown()
		},
		Bind:             []interface{}{},
		BackgroundColour: options.NewRGB(255, 255, 255),
		Debug: options.Debug{
			OpenInspectorOnStartup: true,
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running Wails: %v\n", err)
		os.Exit(1)
	}
}
