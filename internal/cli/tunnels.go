package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

const defaultFoundryPort = 30000

func newLsCmd(d *Deps) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List saved connections and their state",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := d.connect(false)
			if err != nil {
				return err
			}
			defer s.Close()

			tunnels, err := s.Tunnels(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				if tunnels == nil {
					tunnels = []Tunnel{}
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(tunnels)
			}
			if len(tunnels) == 0 {
				return println(cmd, i18n.T("cli_no_tunnels"))
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				i18n.T("tunnel_name"), i18n.T("provider_label"), i18n.T("port"), i18n.T("status_label"), i18n.T("url_label"))
			for _, t := range tunnels {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\n", t.Name, i18n.ProviderText(t.Provider), t.Port, stateLabel(t.State), dash(t.PublicURL))
			}
			return tw.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print the connections as JSON")
	return cmd
}

func newAddCmd(d *Deps) *cobra.Command {
	var provider string
	var port int
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Save a new connection",
		Long: "Save a new connection. Without a name, provider or port on a terminal, ftm asks " +
			"for them; otherwise the provider defaults to cloudflared and the port to 30000.",
		Example: "  ftm add \"Curse of Strahd\" --provider pinggy --port 30000",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := TunnelInput{Provider: provider, LocalPort: port}
			if len(args) == 1 {
				in.Name = strings.TrimSpace(args[0])
			}

			missing := in.Name == "" || !cmd.Flags().Changed("provider") || !cmd.Flags().Changed("port")
			if missing && d.Interactive() {
				if err := tunnelForm(cmd, &in); err != nil {
					return err
				}
			}
			if in.Name == "" {
				return Tr("cli_name_required")
			}
			if in.Provider == "" {
				in.Provider = string(core.ProviderCloudflared)
			}
			if in.LocalPort == 0 {
				in.LocalPort = defaultFoundryPort
			}
			var err error
			if in.Provider, err = checkProvider(in.Provider); err != nil {
				return err
			}
			if _, err := checkPort(strconv.Itoa(in.LocalPort)); err != nil {
				return err
			}

			s, err := d.connect(false)
			if err != nil {
				return err
			}
			defer s.Close()
			t, err := s.CreateTunnel(cmd.Context(), in)
			if err != nil {
				return err
			}
			return println(cmd, i18n.TF("cli_added", t.Name, i18n.ProviderText(t.Provider), t.Port))
		},
	}
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Tunnel provider: "+strings.Join(providerIDs(), ", "))
	cmd.Flags().IntVar(&port, "port", 0, "Port Foundry listens on (default 30000)")
	_ = cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	return cmd
}

// tunnelForm asks for a connection's name, provider and port, starting from
// what in already holds.
func tunnelForm(cmd *cobra.Command, in *TunnelInput) error {
	if in.Provider == "" {
		in.Provider = string(core.ProviderCloudflared)
	}
	if in.LocalPort == 0 {
		in.LocalPort = defaultFoundryPort
	}
	portText := strconv.Itoa(in.LocalPort)

	err := runForm(cmd, huh.NewGroup(
		huh.NewInput().Title(i18n.T("tunnel_name")).Placeholder(i18n.T("tunnel_name_hint")).Value(&in.Name).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return Tr("cli_name_required")
				}
				return nil
			}),
		huh.NewSelect[string]().Title(i18n.T("provider_label")).Options(providerOptions()...).Value(&in.Provider),
		huh.NewInput().Title(i18n.T("port")).Value(&portText).Validate(func(s string) error {
			_, err := checkPort(s)
			return err
		}),
	))
	if err != nil {
		return err
	}
	in.Name = strings.TrimSpace(in.Name)
	in.LocalPort, _ = strconv.Atoi(strings.TrimSpace(portText))
	return nil
}

func newEditCmd(d *Deps) *cobra.Command {
	var name, provider string
	var port int
	cmd := &cobra.Command{
		Use:               "edit <connection>",
		Short:             "Change a saved connection",
		Long:              "Change a saved connection. Without flags on a terminal, ftm asks for the new values.",
		Example:           "  ftm edit \"Friday game\" --port 30001",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: tunnelNames(d),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := d.connect(false)
			if err != nil {
				return err
			}
			defer s.Close()

			tunnels, err := s.Tunnels(cmd.Context())
			if err != nil {
				return err
			}
			t, err := findTunnel(tunnels, args[0])
			if err != nil {
				return err
			}

			in := TunnelInput{Name: name, Provider: provider, LocalPort: port}
			if cmd.Flags().NFlag() == 0 {
				if !d.Interactive() {
					return Tr("cli_edit_needs_flags")
				}
				in = TunnelInput{Name: t.Name, Provider: t.Provider, LocalPort: t.Port}
				if err := tunnelForm(cmd, &in); err != nil {
					return err
				}
			}
			if in.Provider != "" {
				if in.Provider, err = checkProvider(in.Provider); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("port") {
				if _, err := checkPort(strconv.Itoa(in.LocalPort)); err != nil {
					return err
				}
			}

			updated, err := s.UpdateTunnel(cmd.Context(), t.ID, in)
			if err != nil {
				return err
			}
			msg := i18n.TF("cli_edited", updated.Name, i18n.ProviderText(updated.Provider), updated.Port)
			if isRunning(t.State) {
				msg += "\n" + i18n.T("cli_edit_restart_hint")
			}
			return println(cmd, msg)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "New provider")
	cmd.Flags().IntVar(&port, "port", 0, "New Foundry port")
	_ = cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	return cmd
}

func newRmCmd(d *Deps) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:               "rm [connection...]",
		Aliases:           []string{"remove"},
		Short:             "Remove saved connections",
		Long:              "Stop and remove saved connections. ftm asks first unless you pass --yes.",
		ValidArgsFunction: tunnelNames(d),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := d.connect(false)
			if err != nil {
				return err
			}
			defer s.Close()

			tunnels, err := s.Tunnels(cmd.Context())
			if err != nil {
				return err
			}
			targets, err := resolveTunnels(cmd, d, tunnels, args, i18n.T("cli_pick_remove"), true)
			if err != nil {
				return err
			}
			if len(targets) == 0 {
				return nil
			}
			if !yes {
				ok, err := confirm(cmd, d, i18n.TF("cli_confirm_remove", names(targets)))
				if err != nil || !ok {
					return err
				}
			}
			for _, t := range targets {
				if err := s.DeleteTunnel(cmd.Context(), t.ID); err != nil {
					return err
				}
				if err := println(cmd, i18n.TF("cli_removed", t.Name)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Remove without asking")
	return cmd
}

func names(tunnels []Tunnel) string {
	var out []string
	for _, t := range tunnels {
		out = append(out, t.Name)
	}
	return strings.Join(out, ", ")
}

func newUpCmd(d *Deps) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:     "up [connection...]",
		Aliases: []string{"start"},
		Short:   "Open connections and print their links",
		Long: "Open connections and wait until each is online or fails. When no other ftm is " +
			"running, this one keeps them open until you press Ctrl+C.",
		Example:           "  ftm up \"Friday game\"\n  ftm up --all",
		ValidArgsFunction: tunnelNames(d),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := d.connect(true)
			if err != nil {
				return err
			}
			defer s.Close()

			tunnels, err := s.Tunnels(cmd.Context())
			if err != nil {
				return err
			}
			targets := tunnels
			if !all {
				if targets, err = resolveTunnels(cmd, d, tunnels, args, i18n.T("cli_pick_up"), true); err != nil {
					return err
				}
			}
			if len(targets) == 0 {
				return Tr("cli_no_tunnels")
			}

			opened, failed := 0, 0
			for _, t := range targets {
				online, err := openTunnel(cmd, s, t)
				if err != nil {
					return err
				}
				if online.State == string(core.TunnelStateOnline) {
					opened++
					if err := println(cmd, i18n.TF("cli_online", online.Name, online.PublicURL)); err != nil {
						return err
					}
				} else {
					failed++
					printErr(cmd, failure(online))
				}
			}
			if opened > 0 {
				if err := hold(cmd, s); err != nil {
					return err
				}
			}
			if failed > 0 {
				return exitCode(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Open every saved connection")
	return cmd
}

// openTunnel starts a tunnel and reports its states until it is online or
// has failed.
func openTunnel(cmd *cobra.Command, s *session, t Tunnel) (Tunnel, error) {
	if t.State == string(core.TunnelStateOnline) {
		return t, nil
	}
	ctx := cmd.Context()
	var startErr error
	first := true
	last := ""
	result, err := s.WaitTunnel(ctx, t.ID, func(cur Tunnel) bool {
		// The first call comes once state changes are being watched, so
		// starting here cannot miss one.
		if first {
			first = false
			startErr = s.StartTunnel(ctx, t.ID)
			return startErr != nil
		}
		if cur.State != last {
			last = cur.State
			if !settled(cur) {
				printErr(cmd, i18n.TF("cli_state", cur.Name, stateLabel(cur.State)))
			}
		}
		return settled(cur)
	})
	if startErr != nil {
		return result, startErr
	}
	return result, err
}

func failure(t Tunnel) string {
	reason := t.ErrorMessage
	if reason == "" {
		reason = stateLabel(t.State)
	}
	return i18n.TF("cli_failed", t.Name, reason)
}

// hold keeps the tunnels open while this process owns them: they close with
// it. Against a running ftm it returns at once.
func hold(cmd *cobra.Command, s *session) error {
	if s.core == nil {
		return nil
	}
	if url, err := s.core.ServeWeb(0); err == nil {
		printErr(cmd, i18n.TF("dashboard_url", url))
	}
	printErr(cmd, i18n.T("cli_holding"))
	return wait(cmd.Context(), s.core)
}

// wait blocks until the command is interrupted, or relaunches ftm when an
// update asked for it.
func wait(ctx context.Context, c Core) error {
	select {
	case <-ctx.Done():
		return nil
	case <-c.Restart():
		c.Shutdown()
		return c.Relaunch()
	}
}

func newDownCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "down [connection...]",
		Aliases:           []string{"stop"},
		Short:             "Close open connections",
		Long:              "Close the named connections, or every open one when none is named.",
		ValidArgsFunction: tunnelNames(d),
		RunE: func(cmd *cobra.Command, args []string) error {
			if d.Attach == nil {
				return println(cmd, i18n.T("cli_nothing_open"))
			}
			c, err := d.Attach()
			if err != nil {
				return err
			}
			if c == nil {
				// Tunnels live only as long as the ftm that opened them.
				return println(cmd, i18n.T("cli_nothing_open"))
			}
			s := &session{Backend: Remote(c), client: c}

			tunnels, err := s.Tunnels(cmd.Context())
			if err != nil {
				return err
			}
			var targets []Tunnel
			if len(args) == 0 {
				for _, t := range tunnels {
					if isRunning(t.State) {
						targets = append(targets, t)
					}
				}
			} else if targets, err = resolveTunnels(cmd, d, tunnels, args, "", true); err != nil {
				return err
			}
			if len(targets) == 0 {
				return println(cmd, i18n.T("cli_nothing_open"))
			}
			for _, t := range targets {
				if err := s.StopTunnel(cmd.Context(), t.ID); err != nil {
					return err
				}
				if err := println(cmd, i18n.TF("cli_closed", t.Name)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

func newShareCmd(d *Deps) *cobra.Command {
	var port int
	var provider string
	var noCopy bool
	cmd := &cobra.Command{
		Use:   "share [connection]",
		Short: "Open a connection and hand out the invite",
		Long: "Open a connection, copy the link and print a message to send your players. " +
			"With --port and no saved connection on that port, ftm saves one first.",
		Example:           "  ftm share\n  ftm share --port 30000 --provider cloudflared",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: tunnelNames(d),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := d.connect(true)
			if err != nil {
				return err
			}
			defer s.Close()

			t, err := shareTarget(cmd, d, s, args, port, provider)
			if err != nil {
				return err
			}
			online, err := openTunnel(cmd, s, t)
			if err != nil {
				return err
			}
			if online.State != string(core.TunnelStateOnline) {
				printErr(cmd, failure(online))
				return exitCode(1)
			}

			invite := strings.NewReplacer("{name}", online.Name, "{url}", online.PublicURL).Replace(i18n.T("invite_message"))
			out := []string{i18n.TF("cli_online", online.Name, online.PublicURL), "", invite, ""}
			if !noCopy {
				if err := d.Copy(online.PublicURL); err == nil {
					out = append(out, i18n.T("cli_link_copied"))
				}
			}
			for _, line := range out {
				if err := println(cmd, line); err != nil {
					return err
				}
			}
			return hold(cmd, s)
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "Share the connection on this Foundry port, saving one if needed")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider for a connection saved by --port (default cloudflared)")
	cmd.Flags().BoolVar(&noCopy, "no-copy", false, "Don't copy the link")
	_ = cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	return cmd
}

// shareTarget picks the connection to share: the named one, the one on
// --port (saved when there is none), or the one the user picks.
func shareTarget(cmd *cobra.Command, d *Deps, s *session, args []string, port int, provider string) (Tunnel, error) {
	tunnels, err := s.Tunnels(cmd.Context())
	if err != nil {
		return Tunnel{}, err
	}
	if len(args) == 1 {
		return findTunnel(tunnels, args[0])
	}
	if port == 0 {
		picked, err := pickTunnels(cmd, d, tunnels, i18n.T("cli_pick_share"), false)
		if err != nil {
			return Tunnel{}, err
		}
		if len(picked) == 0 {
			return Tunnel{}, Tr("cli_no_tunnels")
		}
		return picked[0], nil
	}
	if _, err := checkPort(strconv.Itoa(port)); err != nil {
		return Tunnel{}, err
	}
	for _, t := range tunnels {
		if t.Port == port && (provider == "" || t.Provider == provider) {
			return t, nil
		}
	}
	if provider == "" {
		provider = string(core.ProviderCloudflared)
	}
	if provider, err = checkProvider(provider); err != nil {
		return Tunnel{}, err
	}
	return s.CreateTunnel(cmd.Context(), TunnelInput{
		Name:      i18n.TF("cli_share_default_name", port),
		Provider:  provider,
		LocalPort: port,
	})
}

func newLogsCmd(d *Deps) *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:               "logs <connection>",
		Short:             "Print a connection's recent output",
		Example:           "  ftm logs \"Friday game\" -f",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: tunnelNames(d),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := d.connect(false)
			if err != nil {
				return err
			}
			defer s.Close()

			tunnels, err := s.Tunnels(cmd.Context())
			if err != nil {
				return err
			}
			t, err := findTunnel(tunnels, args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if !follow {
				lines, err := s.Logs(cmd.Context(), t.ID)
				if err != nil {
					return err
				}
				for _, l := range lines {
					fmt.Fprintln(out, l)
				}
				return nil
			}
			if !s.Attached() {
				printErr(cmd, i18n.T("cli_logs_follow_local"))
			}
			err = s.FollowLogs(cmd.Context(), t.ID, func(line string) { fmt.Fprintln(out, line) })
			if errors.Is(err, context.Canceled) || cmd.Context().Err() != nil {
				return nil
			}
			return err
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Keep printing new lines")
	return cmd
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
