package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/supervisor"
)

// Providers runs `ftm providers [update [id]]`.
func Providers(args []string) error {
	manager := supervisor.NewManager()
	ctx := context.Background()

	switch {
	case len(args) == 0 || args[0] == "list":
		return listProviders(ctx, os.Stdout, manager)
	case args[0] == "update" && len(args) <= 2:
		var id string
		if len(args) == 2 {
			id = args[1]
		}
		return updateProviders(ctx, os.Stdout, manager, id)
	default:
		return errors.New(i18n.T("providers_usage"))
	}
}

func listProviders(ctx context.Context, out io.Writer, manager *supervisor.Manager) error {
	manager.CheckLatest(ctx, true)
	report := manager.ProviderBinaries(ctx)

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
		i18n.T("providers_col_provider"),
		i18n.T("providers_col_installed"),
		i18n.T("providers_col_latest"),
		i18n.T("providers_col_source"))

	var outdated, external []string
	var failures []string
	for _, b := range report {
		marker := ""
		if b.HasUpdate {
			marker = " ↑"
			if b.Updatable {
				outdated = append(outdated, string(b.ID))
			} else {
				external = append(external, i18n.TF("providers_external_hint", b.ID, b.Path))
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", b.ID, dash(b.Version)+marker, dash(b.Latest), sourceLabel(b.Source))

		if b.Error != "" && b.Source != supervisor.SourceSystem {
			failures = append(failures, i18n.TF("providers_check_failed", b.ID, b.Error))
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

// updateProviders updates one provider, or every bundled one that is behind
// when id is empty.
func updateProviders(ctx context.Context, out io.Writer, manager *supervisor.Manager, id string) error {
	var targets []core.Provider

	if id != "" {
		provider := core.Provider(strings.ToLower(id))
		if !isKnownProvider(provider) {
			return errors.New(i18n.TF("providers_unknown", id))
		}
		info := manager.ProviderBinary(ctx, provider)
		switch info.Source {
		case supervisor.SourceSystem:
			return errors.New(i18n.TF("providers_not_updatable", provider))
		case supervisor.SourcePath:
			return errors.New(i18n.TF("providers_external", provider, info.Path))
		}
		targets = append(targets, provider)
	} else {
		manager.CheckLatest(ctx, true)
		for _, b := range manager.ProviderBinaries(ctx) {
			if b.HasUpdate && b.Updatable {
				targets = append(targets, b.ID)
			}
		}
		if len(targets) == 0 {
			fmt.Fprintln(out, i18n.T("providers_all_up_to_date"))
			return nil
		}
	}

	var failures []error
	for _, provider := range targets {
		fmt.Fprintln(out, i18n.TF("providers_updating", provider))

		progress, done := printProgress(out)
		result, err := manager.UpdateProvider(ctx, provider, progress)
		close(progress)
		<-done

		if err != nil {
			failures = append(failures, errors.New(i18n.TF("providers_update_failed", provider, err)))
			continue
		}
		fmt.Fprintln(out, i18n.TF("providers_updated", provider, dash(result.Version)))
	}

	return errors.Join(failures...)
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
func printProgress(out io.Writer) (chan provider.DownloadProgress, <-chan struct{}) {
	ch := make(chan provider.DownloadProgress, 16)
	done := make(chan struct{})

	go func() {
		defer close(done)
		last := -1
		drew := false
		for p := range ch {
			pct := int(p.Percent)
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

func sourceLabel(s supervisor.BinarySource) string {
	switch s {
	case supervisor.SourceBundled:
		return i18n.T("providers_source_bundled")
	case supervisor.SourcePath:
		return i18n.T("providers_source_path")
	case supervisor.SourceSystem:
		return i18n.T("providers_source_system")
	default:
		return i18n.T("providers_source_none")
	}
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
