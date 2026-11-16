package main

import (
	"fmt"
	"net/url"

	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const pipWindowName = "ftm-pip"

func miniURL(webURL, tunnelID string) string {
	return fmt.Sprintf("%s/mini?id=%s", webURL, url.QueryEscape(tunnelID))
}

func openPiPWindow(wailsApp *application.App, webURL, tunnelID string) {
	target := miniURL(webURL, tunnelID)

	if win, ok := wailsApp.Window.GetByName(pipWindowName); ok {
		win.SetURL(target)
		win.Restore()
		win.Show()
		win.Focus()
		return
	}

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             pipWindowName,
		Title:            i18n.T("pip_title"),
		Width:            340,
		Height:           220,
		MinWidth:         300,
		MinHeight:        180,
		AlwaysOnTop:      true,
		URL:              target,
		BackgroundColour: application.NewRGB(255, 255, 255),
		DevToolsEnabled:  false,
		Mac: application.MacWindow{
			TitleBar:    application.MacTitleBarHiddenInset,
			WindowLevel: application.MacWindowLevelFloating,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
			DisableIcon:     true,
			DisableMenu:     true,
		},
	})
}
