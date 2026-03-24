package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/deadbryam/ftm/internal/app"
	"github.com/deadbryam/ftm/internal/version"
)

var BuildVersion string

func main() {
	var (
		webOnly     = flag.Bool("web", false, "Start web dashboard and open browser")
		server      = flag.Bool("server", false, "Start web dashboard only (no browser)")
		port        = flag.Int("port", 0, "Web server port (auto-detect if not specified)")
		showVersion = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Foundry Tunnel Manager v%s\n", version.Version)
		os.Exit(0)
	}

	if BuildVersion == "" {
		BuildVersion = version.Version
	}

	application, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *port > 0 {
		application.Config.WebPort = *port
	}

	if err := application.StartWebServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
		os.Exit(1)
	}

	url := application.WebServer.URL()
	fmt.Printf("🎲 Foundry Tunnel Manager v%s\n", BuildVersion)
	fmt.Printf("🌐 Dashboard running at: %s\n", url)

	if *webOnly {
		fmt.Println("\nPress Ctrl+C to stop")
		application.OpenDashboard()
		select {}
	} else if *server {
		fmt.Println("\nPress Ctrl+C to stop")
		select {}
	}

	fmt.Printf("\nPress 'w' in the TUI to open the dashboard\n\n")

	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
