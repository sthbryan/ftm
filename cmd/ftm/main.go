package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sthbryan/ftm/internal/app"
	"github.com/sthbryan/ftm/internal/cli"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/platform/instance"
	"github.com/sthbryan/ftm/internal/version"
)

func main() {
	if err := cli.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "providers" {
		if err := cli.Providers(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	var (
		webOnly     = flag.Bool("web", false, "Start web dashboard and open browser")
		server      = flag.Bool("server", false, "Start web dashboard only (no browser)")
		port        = flag.Int("port", 0, "Web server port (auto-detect if not specified)")
		showVersion = flag.Bool("version", false, "Show version")
		uninstall   = flag.Bool("uninstall", false, "Uninstall ftm")
		update      = flag.Bool("update", false, "Update ftm to the latest release")
		checkOnly   = flag.Bool("check", false, "Check for updates without installing")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(i18n.TF("version_output", version.Version))
		os.Exit(0)
	}

	if *uninstall {
		if err := cli.Uninstall(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *checkOnly {
		if err := cli.Update(true); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *update {
		if err := cli.Update(false); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	application, err := app.New()
	if err != nil {
		var running *instance.RunningError
		if errors.As(err, &running) {
			os.Exit(attach(running.Info, *webOnly, *server))
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *port > 0 {
		if err := application.SetWebPort(*port); err != nil {
			fmt.Fprintf(os.Stderr, "Could not save the web port: %v\n", err)
		}
	}

	if err := application.StartWebServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
		os.Exit(1)
	}

	url := application.WebServer.URL()
	fmt.Printf("🎲 Foundry Tunnel Manager v%s\n", version.Version)
	fmt.Print(i18n.TF("dashboard_url", url))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	if *webOnly || *server {
		fmt.Print(i18n.T("press_ctrl_c"))
		if *webOnly {
			if err := application.OpenDashboard(); err != nil {
				fmt.Fprintf(os.Stderr, "Could not open the dashboard: %v\n", err)
			}
		}

		<-stop
		application.Shutdown()
		return
	}

	go func() {
		<-stop
		application.Shutdown()
		os.Exit(130)
	}()

	fmt.Print(i18n.T("tui_hint"))

	err = application.Run()
	application.Shutdown()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// attach runs when another ftm or ftm-desktop already owns the tunnels: it
// points at that process's dashboard instead of starting a second manager that
// would fight it over the config and the port. It returns the exit code.
func attach(info *instance.Info, openBrowser, headless bool) int {
	if info == nil {
		fmt.Fprintln(os.Stderr, i18n.T("already_starting"))
		return 1
	}

	fmt.Println(i18n.TF("already_running", info.URL()))

	if openBrowser {
		if err := app.OpenURL(info.URL()); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open the dashboard: %v\n", err)
			return 1
		}
		return 0
	}

	if !headless {
		fmt.Println(i18n.T("already_running_tui"))
	}
	return 1
}
