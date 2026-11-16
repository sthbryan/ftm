package cli

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

func println(cmd *cobra.Command, s string) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), s)
	return err
}

func printErr(cmd *cobra.Command, s string) {
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), s)
}

// errAutostartUnsupported is returned when start at login is switched on a
// build that does not have it.
var errAutostartUnsupported = Tr("cli_autostart_unsupported")

// Tr returns an error whose message is the translation of key.
func Tr(key string) error { return translatedError(key) }

// translatedError is an error whose message is a translation key, looked up
// when it is printed so it follows the language picked at start.
type translatedError string

func (e translatedError) Error() string { return i18n.T(string(e)) }

// stateLabel is how a tunnel state reads in a table and in progress lines.
func stateLabel(state string) string {
	switch core.TunnelState(state) {
	case core.TunnelStateStarting:
		return i18n.T("status_starting")
	case core.TunnelStateConnecting:
		return i18n.T("status_connecting")
	case core.TunnelStateOnline:
		return i18n.T("status_online")
	case core.TunnelStateError:
		return i18n.T("status_error")
	case core.TunnelStateTimeout:
		return i18n.T("status_timeout")
	case core.TunnelStateStopping:
		return i18n.T("stopping")
	case core.TunnelStateInstall:
		return i18n.T("installing")
	case core.TunnelStateDownload:
		return i18n.T("downloading")
	case stateNeedsInstall:
		return i18n.T("need_installing")
	}
	return i18n.T("status_offline")
}

// isRunning is true for a tunnel that is on its way up or up.
func isRunning(state string) bool {
	switch core.TunnelState(state) {
	case core.TunnelStateStarting, core.TunnelStateConnecting, core.TunnelStateOnline,
		core.TunnelStateInstall, core.TunnelStateDownload:
		return true
	}
	return false
}

// settled is true once a starting tunnel is online or has given up.
func settled(t Tunnel) bool {
	switch core.TunnelState(t.State) {
	case core.TunnelStateOnline, core.TunnelStateError, core.TunnelStateTimeout, core.TunnelStateStopped:
		return true
	}
	return false
}

// findTunnel matches ref against ids first, then names, ignoring case.
func findTunnel(tunnels []Tunnel, ref string) (Tunnel, error) {
	for _, t := range tunnels {
		if t.ID == ref {
			return t, nil
		}
	}
	var matches []Tunnel
	for _, t := range tunnels {
		if strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(ref)) {
			matches = append(matches, t)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Tunnel{}, errors.New(i18n.TF("cli_tunnel_not_found", ref))
	}
	return Tunnel{}, errors.New(i18n.TF("cli_tunnel_ambiguous", ref))
}

// providerIDs are the provider ids in their usual order.
func providerIDs() []string {
	var ids []string
	for _, p := range core.AllProviders() {
		ids = append(ids, string(p))
	}
	return ids
}

func checkProvider(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !slices.Contains(providerIDs(), id) {
		return "", errors.New(i18n.TF("cli_provider_unknown", id, strings.Join(providerIDs(), ", ")))
	}
	return id, nil
}

func checkPort(raw string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New(i18n.T("cli_port_invalid"))
	}
	return port, nil
}

func providerOptions() []huh.Option[string] {
	var opts []huh.Option[string]
	for _, id := range providerIDs() {
		opts = append(opts, huh.NewOption(i18n.ProviderText(id), id))
	}
	return opts
}

// runForm shows a Huh form on the command's terminal. A form the user
// closes ends the command without an error message.
func runForm(cmd *cobra.Command, groups ...*huh.Group) error {
	err := huh.NewForm(groups...).
		WithInput(cmd.InOrStdin()).
		WithOutput(cmd.OutOrStdout()).
		RunWithContext(cmd.Context())
	if errors.Is(err, huh.ErrUserAborted) {
		return exitCode(130)
	}
	return err
}

// pickTunnels asks which tunnels to act on. With one saved tunnel it is the
// answer; without a terminal the user has to name them.
func pickTunnels(cmd *cobra.Command, d *Deps, tunnels []Tunnel, title string, multi bool) ([]Tunnel, error) {
	switch {
	case len(tunnels) == 0:
		return nil, errors.New(i18n.T("cli_no_tunnels"))
	case len(tunnels) == 1:
		return tunnels, nil
	case !d.Interactive():
		return nil, errors.New(i18n.T("cli_name_a_tunnel"))
	}

	var opts []huh.Option[string]
	for _, t := range tunnels {
		opts = append(opts, huh.NewOption(fmt.Sprintf("%s  (%s, %d)", t.Name, i18n.ProviderText(t.Provider), t.Port), t.ID))
	}

	var chosen []string
	if multi {
		if err := runForm(cmd, huh.NewGroup(huh.NewMultiSelect[string]().Title(title).Options(opts...).Value(&chosen))); err != nil {
			return nil, err
		}
	} else {
		var one string
		if err := runForm(cmd, huh.NewGroup(huh.NewSelect[string]().Title(title).Options(opts...).Value(&one))); err != nil {
			return nil, err
		}
		chosen = []string{one}
	}

	var out []Tunnel
	for _, t := range tunnels {
		if slices.Contains(chosen, t.ID) {
			out = append(out, t)
		}
	}
	return out, nil
}

// resolveTunnels turns references into tunnels, or asks when there are none.
func resolveTunnels(cmd *cobra.Command, d *Deps, tunnels []Tunnel, refs []string, title string, multi bool) ([]Tunnel, error) {
	if len(refs) == 0 {
		return pickTunnels(cmd, d, tunnels, title, multi)
	}
	var out []Tunnel
	for _, ref := range refs {
		t, err := findTunnel(tunnels, ref)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// confirm asks a yes/no question. Without a terminal the user has to pass
// --yes instead.
func confirm(cmd *cobra.Command, d *Deps, title string) (bool, error) {
	if !d.Interactive() {
		return false, errors.New(i18n.T("cli_confirm_with_yes"))
	}
	ok := false
	err := runForm(cmd, huh.NewGroup(huh.NewConfirm().Title(title).Affirmative(i18n.T("cli_yes")).Negative(i18n.T("cancel")).Value(&ok)))
	return ok, err
}

// tunnelNames lists saved tunnel names for shell completion.
func tunnelNames(d *Deps) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		if d.Attach == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		c, err := d.Attach()
		var names []string
		if err == nil && c != nil {
			if tunnels, err := c.Tunnels(cmd.Context()); err == nil {
				for _, t := range tunnels {
					names = append(names, t.Name)
				}
			}
		} else if d.Peek != nil {
			if cfg, err := d.Peek(); err == nil {
				for _, t := range cfg.Tunnels {
					names = append(names, t.Name)
				}
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeProviders(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return providerIDs(), cobra.ShellCompDirectiveNoFileComp
}
