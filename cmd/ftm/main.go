package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sthbryan/ftm/internal/app"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/version"
)

var BuildVersion string

func doUninstall() {
	binaryPath, err := exec.LookPath("ftm")
	if err != nil {
		fmt.Println(i18n.T("uninstall_not_found"))
		os.Exit(1)
	}

	absPath, err := filepath.EvalSymlinks(binaryPath)
	if err != nil {
		absPath = binaryPath
	}

	fmt.Println(i18n.TF("uninstall_removing", absPath))
	if err := os.Remove(absPath); err != nil {
		fmt.Fprintf(os.Stderr, i18n.TF("uninstall_error", err.Error())+"\n")
		os.Exit(1)
	}

	fmt.Println(i18n.T("uninstall_success"))
}

func main() {
	var (
		webOnly     = flag.Bool("web", false, "Start web dashboard and open browser")
		server      = flag.Bool("server", false, "Start web dashboard only (no browser)")
		port        = flag.Int("port", 0, "Web server port (auto-detect if not specified)")
		showVersion = flag.Bool("version", false, "Show version")
		uninstall   = flag.Bool("uninstall", false, "Uninstall ftm")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(i18n.TF("version_output", version.Version))
		os.Exit(0)
	}

	if *uninstall {
		doUninstall()
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
	fmt.Printf(i18n.TF("dashboard_url", url))

	if *webOnly {
		fmt.Print(i18n.T("press_ctrl_c"))
		application.OpenDashboard()
		select {}
	} else if *server {
		fmt.Print(i18n.T("press_ctrl_c"))
		select {}
	}

	fmt.Print(i18n.T("tui_hint"))

	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
