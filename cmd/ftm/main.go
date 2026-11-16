// Command ftm is Foundry Tunnel Manager's command line and terminal view. It
// wires the store, the supervisor and the bus into a core and hands it to
// internal/cli, which runs the command.
package main

import (
	"context"
	"os"

	"github.com/sthbryan/ftm/internal/cli"
	"github.com/sthbryan/ftm/internal/client"
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/platform/autostart"
	"github.com/sthbryan/ftm/internal/platform/clipboard"
	"github.com/sthbryan/ftm/internal/platform/open"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/ui/tui"
	"github.com/sthbryan/ftm/internal/updater"
)

func main() {
	os.Exit(cli.Execute(context.Background(), os.Args[1:], deps()))
}

func deps() cli.Deps {
	dirs := store.DefaultDirs()
	return cli.Deps{
		Dirs: cli.Dirs{
			Config:     dirs.Config,
			ConfigFile: dirs.ConfigFile(),
			Bin:        dirs.Bin,
			Runtime:    dirs.Runtime,
			LogFile:    dirs.LogFile(),
		},
		Peek:   peekConfig,
		Attach: func() (*client.Client, error) { return client.Find(dirs.Runtime) },
		AttachTUI: func(c *client.Client) error {
			if file := redirectLog(); file != nil {
				defer file.Close()
			}
			_, err := tui.Run(tui.NewRemote(c, dirs.Config, func() (*client.Client, error) {
				return client.Find(dirs.Runtime)
			}))
			return err
		},
		Open: func(background bool) (cli.Core, error) {
			if !background {
				// A quick command's output is its own; the core's log goes
				// to the log file.
				redirectLog()
			}
			app, err := newApp(background)
			if err != nil {
				return nil, err
			}
			return appCore{app}, nil
		},
		Autostart: autostart.New(),
		OpenURL:   open.URL,
		Copy:      clipboard.Write,
	}
}

// peekConfig reads the config without writing it, since another ftm may own
// it.
func peekConfig() (core.Config, error) {
	return store.Peek(store.DefaultDirs())
}

// appCore is the App as the command tree sees it.
type appCore struct{ app *App }

func (c appCore) Service() *service.Service { return c.app.Service }
func (c appCore) Bus() *events.Bus          { return c.app.Bus }
func (c appCore) RunTUI() error             { return c.app.Run() }
func (c appCore) Restart() <-chan struct{}  { return c.app.RestartRequested() }
func (c appCore) Relaunch() error           { return updater.Relaunch() }
func (c appCore) Shutdown()                 { c.app.Shutdown() }

func (c appCore) ServeWeb(port int) (string, error) {
	if port > 0 {
		if err := c.app.SetWebPort(port); err != nil {
			return "", err
		}
	}
	if err := c.app.StartWebServer(); err != nil {
		return "", err
	}
	return c.app.WebServer.URL(), nil
}
