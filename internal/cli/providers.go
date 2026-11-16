package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

func newProvidersCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "List, download and update the tunnel providers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withBackend(d, func(b Backend) error { return listProviders(cmd.Context(), cmd.OutOrStdout(), b) })
		},
	}

	ls := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List each provider's version and where it comes from",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withBackend(d, func(b Backend) error { return listProviders(cmd.Context(), cmd.OutOrStdout(), b) })
		},
	}
	install := &cobra.Command{
		Use:               "install [provider...]",
		Short:             "Download providers ftm doesn't have yet",
		Long:              "Download the named providers, or every one ftm can download and doesn't have yet.",
		ValidArgsFunction: completeProviders,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBackend(d, func(b Backend) error { return installProviders(cmd.Context(), cmd.OutOrStdout(), b, args) })
		},
	}
	update := &cobra.Command{
		Use:               "update [provider]",
		Short:             "Update ftm's copy of a provider, or every one that is behind",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProviders,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			return withBackend(d, func(b Backend) error { return updateProviders(cmd.Context(), cmd.OutOrStdout(), b, id) })
		},
	}
	cmd.AddCommand(ls, install, update)
	return cmd
}

func withBackend(d *Deps, fn func(Backend) error) error {
	s, err := d.connect(false)
	if err != nil {
		return err
	}
	defer s.Close()
	return fn(s)
}

func listProviders(ctx context.Context, out io.Writer, b Backend) error {
	report, err := b.Providers(ctx, true)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
		i18n.T("providers_col_provider"),
		i18n.T("providers_col_installed"),
		i18n.T("providers_col_latest"),
		i18n.T("providers_col_source"))

	var outdated, external, failures []string
	for _, p := range report {
		marker := ""
		if p.HasUpdate {
			marker = " ↑"
			if p.Updatable {
				outdated = append(outdated, p.ID)
			} else {
				external = append(external, i18n.TF("providers_external_hint", p.ID, p.Path))
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.ID, dash(p.Version)+marker, dash(p.Latest), sourceLabel(p.Source))

		if p.Error != "" && p.Source != string(core.SourceSystem) {
			failures = append(failures, i18n.TF("providers_check_failed", p.ID, p.Error))
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(out)
	for _, f := range failures {
		fmt.Fprintln(out, f)
	}
	for _, e := range external {
		fmt.Fprintln(out, e)
	}
	if len(outdated) > 0 {
		fmt.Fprintln(out, i18n.TF("providers_updates_available", strings.Join(outdated, ", ")))
	} else if len(external) == 0 && len(failures) == 0 {
		fmt.Fprintln(out, i18n.T("providers_all_up_to_date"))
	}
	return nil
}

// installProviders downloads the named providers, or every downloadable one
// that is missing.
func installProviders(ctx context.Context, out io.Writer, b Backend, ids []string) error {
	report, err := b.Providers(ctx, false)
	if err != nil {
		return err
	}
	byID := map[string]ProviderBinary{}
	for _, p := range report {
		byID[p.ID] = p
	}

	var targets []string
	if len(ids) == 0 {
		for _, p := range report {
			if !p.Installed && p.Source == string(core.SourceNone) {
				targets = append(targets, p.ID)
			}
		}
		if len(targets) == 0 {
			fmt.Fprintln(out, i18n.T("cli_providers_all_installed"))
			return nil
		}
	}
	for _, raw := range ids {
		id, err := checkProvider(raw)
		if err != nil {
			return err
		}
		if p := byID[id]; p.Installed {
			fmt.Fprintln(out, i18n.TF("cli_provider_installed", id, dash(p.Version)))
			continue
		}
		targets = append(targets, id)
	}
	return runProviderUpdates(ctx, out, b, targets)
}

// updateProviders updates one provider, or every bundled one that is behind
// when id is empty.
func updateProviders(ctx context.Context, out io.Writer, b Backend, id string) error {
	var targets []string

	if id != "" {
		id = strings.ToLower(id)
		if !isKnownProvider(core.Provider(id)) {
			return errors.New(i18n.TF("providers_unknown", id))
		}
		report, err := b.Providers(ctx, false)
		if err != nil {
			return err
		}
		for _, p := range report {
			if p.ID != id {
				continue
			}
			switch core.BinarySource(p.Source) {
			case core.SourceSystem:
				return errors.New(i18n.TF("providers_not_updatable", id))
			case core.SourcePath:
				return errors.New(i18n.TF("providers_external", id, p.Path))
			}
		}
		targets = append(targets, id)
	} else {
		report, err := b.Providers(ctx, true)
		if err != nil {
			return err
		}
		for _, p := range report {
			if p.HasUpdate && p.Updatable {
				targets = append(targets, p.ID)
			}
		}
		if len(targets) == 0 {
			fmt.Fprintln(out, i18n.T("providers_all_up_to_date"))
			return nil
		}
	}
	return runProviderUpdates(ctx, out, b, targets)
}

func runProviderUpdates(ctx context.Context, out io.Writer, b Backend, targets []string) error {
	var failures []error
	for _, id := range targets {
		fmt.Fprintln(out, i18n.TF("providers_updating", id))

		progress, done := printProgress(out)
		version, err := b.UpdateProvider(ctx, id, func(p float64) { progress <- p })
		close(progress)
		<-done

		if err != nil {
			failures = append(failures, providerFailure(id, err))
			continue
		}
		fmt.Fprintln(out, i18n.TF("providers_updated", id, dash(version)))
	}
	return errors.Join(failures...)
}

// providerFailure explains why a provider could not be updated.
func providerFailure(id string, err error) error {
	switch {
	case errors.Is(err, core.ErrNotUpdatable):
		return errors.New(i18n.TF("providers_not_updatable", id))
	case errors.Is(err, core.ErrExternalBinary):
		return errors.New(i18n.TF("providers_external_hint", id, err))
	case errors.Is(err, core.ErrUnknownProvider):
		return errors.New(i18n.TF("providers_unknown", id))
	}
	return errors.New(i18n.TF("providers_update_failed", id, err))
}

func isKnownProvider(p core.Provider) bool {
	for _, known := range core.AllProviders() {
		if known == p {
			return true
		}
	}
	return false
}

// printProgress draws a single updating percentage line until the channel is
// closed.
func printProgress(out io.Writer) (chan float64, <-chan struct{}) {
	ch := make(chan float64, 16)
	done := make(chan struct{})

	go func() {
		defer close(done)
		last := -1
		drew := false
		for p := range ch {
			pct := int(p)
			if pct == last {
				continue
			}
			last = pct
			drew = true
			fmt.Fprintf(out, "\r  %3d%%", pct)
		}
		if drew {
			fmt.Fprintln(out)
		}
	}()

	return ch, done
}

func sourceLabel(s string) string {
	switch core.BinarySource(s) {
	case core.SourceBundled:
		return i18n.T("providers_source_bundled")
	case core.SourcePath:
		return i18n.T("providers_source_path")
	case core.SourceSystem:
		return i18n.T("providers_source_system")
	default:
		return i18n.T("providers_source_none")
	}
}
