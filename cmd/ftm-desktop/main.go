// Command ftm-desktop is the Wails window around the ftm dashboard. It owns
// the application core (or attaches to the process that does), never opens a
// terminal, and runs once per user: a second launch focuses this window.
package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/sthbryan/ftm/internal/buildinfo"
	"github.com/sthbryan/ftm/internal/platform/autostart"
	"github.com/sthbryan/ftm/internal/platform/instance"
	"github.com/sthbryan/ftm/internal/version"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// desktopID names the window's single-instance lock. It matches the Windows
// AppUserModelID and the MSIX identity so the OS treats them as one app.
const desktopID = "com.justcallmebryan.ftm"

const mainWindowName = "ftm-main"

func main() {
	var port int
	flag.IntVar(&port, "port", 0, "Web server port")
	flag.Parse()

	// Autostart and the updater read this, so it runs before either.
	buildinfo.MarkDesktop()

	if err := autostart.New().Repair(); err != nil {
		log.Printf("Could not refresh the autostart entry: %v", err)
	}

	var ftm *appCore

	// application.New exits a second launch after handing over to this one,
	// before it can take the core lock or open a window.
	wailsApp := application.New(application.Options{
		Name:        "Foundry Tunnel Manager",
		Description: "Share your Foundry VTT world with your players",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: desktopID,
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				application.InvokeAsync(focusMainWindow)
			},
		},
		OnShutdown: func() {
			if ftm != nil {
				ftm.shutdown()
			}
		},
	})

	webURL, ftm, err := openCore(port)
	if err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}

	if ftm != nil {
		ftm.web.SetPiPOpener(func(tunnelID string) error {
			application.InvokeAsync(func() {
				openPiPWindow(wailsApp, webURL, tunnelID)
			})
			return nil
		})
	}

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             mainWindowName,
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
		log.Printf("Error running Wails: %v", err)
		os.Exit(1)
	}
}

// openCore starts the core, or returns the dashboard of the ftm process
// (TUI or `ftm serve`) that already owns it, with a nil core.
func openCore(port int) (string, *appCore, error) {
	c, err := startCore(port)
	var running *instance.RunningError
	switch {
	case errors.As(err, &running):
		if running.Info == nil {
			return "", nil, running
		}
		log.Printf("Attaching to the running instance at %s", running.Info.URL())
		return running.Info.URL(), nil, nil
	case err != nil:
		return "", nil, err
	}

	log.Printf("Foundry Tunnel Manager v%s at %s", version.Version, c.url())
	return c.url(), c, nil
}

// focusMainWindow brings the window back when ftm-desktop is launched again,
// from the Start menu, autostart or a shortcut.
func focusMainWindow() {
	app := application.Get()
	if app == nil {
		return
	}
	if win, ok := app.Window.GetByName(mainWindowName); ok {
		raise(win)
	}
}

// window is the part of a Wails window raise needs.
type window interface {
	Restore()
	Show() application.Window
	Focus()
}

func raise(win window) {
	win.Restore()
	win.Show()
	win.Focus()
}
