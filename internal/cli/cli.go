// Package cli is ftm's command tree. cmd/ftm wires the core and hands it in
// through Deps; every command acts on a running ftm over its API when there
// is one, and on a core started in this process otherwise.
package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"

	"charm.land/fang/v2"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/sthbryan/ftm/internal/client"
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/platform/instance"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/version"
)

// Init loads the translations and switches to the saved language. peek reads
// the config without writing it, since another ftm may own it; when it fails
// the system language is used.
func Init(peek func() (core.Config, error)) error {
	if err := i18n.Load(); err != nil {
		return err
	}

	cfg, err := peek()
	if err != nil {
		i18n.UseLanguage("")
		return nil
	}

	i18n.UseLanguage(cfg.Language)

	return nil
}

// Core is the ftm core run inside this process. cmd/ftm builds it from the
// store, the supervisor and the bus.
type Core interface {
	Service() *service.Service
	Bus() *events.Bus
	// ServeWeb starts the web server, preferring port when it is not zero,
	// and returns its URL. Other processes attach through it.
	ServeWeb(port int) (string, error)
	// RunTUI shows the TUI until the user quits.
	RunTUI() error
	// Restart is closed when an update asks this process to start again.
	Restart() <-chan struct{}
	// Relaunch starts the updated binary in place of this one.
	Relaunch() error
	// Shutdown stops every tunnel and releases the instance lock.
	Shutdown()
}

// Dirs are the paths commands report and check.
type Dirs struct {
	Config     string
	ConfigFile string
	Bin        string
	Runtime    string
	LogFile    string
}

// Deps is what cmd/ftm hands the command tree.
type Deps struct {
	Dirs Dirs
	// Peek reads the config without writing it.
	Peek func() (core.Config, error)
	// Attach returns a client for the running ftm, or nil when there is none.
	Attach func() (*client.Client, error)
	// AttachTUI shows the TUI on the running ftm c talks to. Nil means the
	// TUI only runs on a core of its own.
	AttachTUI func(c *client.Client) error
	// Open runs the core in this process. background starts the periodic
	// work (stats, update checks) that only a long-running command needs.
	// When another process owns the core it returns *instance.RunningError.
	Open func(background bool) (Core, error)
	// Autostart is ftm-desktop's start at login.
	Autostart Autostart
	// Interactive reports whether prompts can be shown. Nil means "when
	// stdin and stdout are terminals".
	Interactive func() bool
	// Open a URL in the browser.
	OpenURL func(string) error
	// Copy text to the clipboard.
	Copy func(string) error
	// Doctor overrides the checks doctor runs; nil means the real ones.
	Doctor *DoctorEnv

	In       io.Reader
	Out, Err io.Writer
}

func (d *Deps) defaults() {
	if d.In == nil {
		d.In = os.Stdin
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
	if d.Err == nil {
		d.Err = os.Stderr
	}
	if d.Interactive == nil {
		d.Interactive = func() bool {
			return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
		}
	}
	if d.OpenURL == nil {
		d.OpenURL = func(string) error { return errors.New("no browser") }
	}
	if d.Copy == nil {
		d.Copy = func(string) error { return errors.New("no clipboard") }
	}
}

// Execute runs the command line in args (without the program name) and
// returns the process exit code.
func Execute(ctx context.Context, args []string, deps Deps) int {
	deps.defaults()
	if deps.Peek != nil {
		if err := Init(deps.Peek); err != nil {
			_, _ = io.WriteString(deps.Err, "Error: "+err.Error()+"\n")
			return 1
		}
	}

	root := newRoot(&deps)
	root.SetArgs(translateLegacy(args))
	root.SetIn(deps.In)
	root.SetOut(deps.Out)
	root.SetErr(deps.Err)

	err := fang.Execute(ctx, root,
		fang.WithVersion(version.Version),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	)
	var exit exitCode
	switch {
	case errors.As(err, &exit):
		return int(exit)
	case err != nil:
		return 1
	}
	return 0
}

// exitCode ends a command with a code and no further message: the command
// already said what happened.
type exitCode int

func (e exitCode) Error() string { return "" }

// session is the ftm a command acts on, plus the core to shut down after it
// when this process runs one.
type session struct {
	Backend
	core   Core
	client *client.Client
}

// Close shuts down the core this process ran, if any.
func (s *session) Close() {
	if s.core != nil {
		s.core.Shutdown()
	}
}

// connect attaches to the running ftm or runs the core here.
func (d *Deps) connect(background bool) (*session, error) {
	if d.Attach != nil {
		c, err := d.Attach()
		if err != nil {
			return nil, err
		}
		if c != nil {
			return &session{Backend: Remote(c), client: c}, nil
		}
	}
	if d.Open == nil {
		return nil, errors.New(i18n.T("cli_no_core"))
	}

	c, err := d.Open(background)
	var running *instance.RunningError
	if errors.As(err, &running) {
		if running.Info == nil {
			return nil, errors.New(i18n.T("already_starting"))
		}
		rc := client.New(running.Info.URL(), running.Info.Token)
		return &session{Backend: Remote(rc), client: rc}, nil
	}
	if err != nil {
		return nil, err
	}
	return &session{Backend: Local(c.Service(), c.Bus(), d.Autostart), core: c}, nil
}

func newRoot(d *Deps) *cobra.Command {
	var legacy legacyFlags

	root := &cobra.Command{
		Use:   "ftm",
		Short: "Share your Foundry VTT game through a tunnel",
		Long: "Foundry Tunnel Manager opens a public link to the Foundry VTT server on this " +
			"computer so your players can join from anywhere.\n\n" +
			"Run ftm on its own to manage connections in the terminal. Every command works " +
			"against the ftm already running (the desktop app or ftm serve) when there is one.",
		Example: "  ftm add \"Friday game\" --provider cloudflared --port 30000\n" +
			"  ftm share \"Friday game\"\n" +
			"  ftm ls --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTUI(cmd, d, legacy.port)
		},
	}
	legacy.register(root)

	root.AddGroup(
		&cobra.Group{ID: groupTunnels, Title: "Connections"},
		&cobra.Group{ID: groupApp, Title: "ftm"},
	)

	for _, c := range []*cobra.Command{
		newUpCmd(d), newDownCmd(d), newShareCmd(d), newLsCmd(d), newAddCmd(d),
		newRmCmd(d), newEditCmd(d), newLogsCmd(d),
	} {
		c.GroupID = groupTunnels
		root.AddCommand(c)
	}
	for _, c := range []*cobra.Command{
		newServeCmd(d), newProvidersCmd(d), newDoctorCmd(d), newConfigCmd(d),
		newSelfUpdateCmd(d), newUninstallCmd(d), newVersionCmd(d),
	} {
		c.GroupID = groupApp
		root.AddCommand(c)
	}
	root.SetHelpCommandGroupID(groupApp)
	root.SetCompletionCommandGroupID(groupApp)
	return root
}

const (
	groupTunnels = "tunnels"
	groupApp     = "app"
)

func newVersionCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return println(cmd, i18n.TF("version_output", version.Version))
		},
	}
}

// runTUI shows the TUI on a core run here. With another ftm running (the
// desktop app or ftm serve) it attaches to that one instead, since two
// managers would fight over the config and the port.
func runTUI(cmd *cobra.Command, d *Deps, port int) error {
	if d.Attach != nil {
		if c, err := d.Attach(); err == nil && c != nil {
			return attachedTUI(cmd, d, c)
		}
	}
	if d.Open == nil {
		return errors.New(i18n.T("cli_no_core"))
	}
	c, err := d.Open(true)
	var running *instance.RunningError
	if errors.As(err, &running) {
		if running.Info == nil {
			return errors.New(i18n.T("already_starting"))
		}
		return attachedTUI(cmd, d, client.New(running.Info.URL(), running.Info.Token))
	}
	if err != nil {
		return err
	}
	defer c.Shutdown()

	if port > 0 {
		if err := c.Service().SetWebPort(port); err != nil {
			printErr(cmd, i18n.TF("cli_port_not_saved", err))
		}
	}
	printErr(cmd, i18n.T("tui_hint"))
	return c.RunTUI()
}

func attachedTUI(cmd *cobra.Command, d *Deps, c *client.Client) error {
	if d.AttachTUI != nil {
		return d.AttachTUI(c)
	}
	_ = println(cmd, i18n.TF("already_running", c.URL()))
	_ = println(cmd, i18n.T("already_running_tui"))
	return exitCode(1)
}
