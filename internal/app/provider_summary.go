package app

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/process"
)

type providerSummaryMsg struct {
	line string
}

const providerSummaryTimeout = 20 * time.Second

// loadProviderSummary reads the provider binaries in the background, since
// it can run each binary's --version and reach the release APIs.
func (m *Model) loadProviderSummary() tea.Cmd {
	manager := m.App.Manager
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), providerSummaryTimeout)
		defer cancel()
		return providerSummaryMsg{line: providerSummary(manager.ProviderBinaries(ctx))}
	}
}

// providerSummary renders the downloaded providers as
// "cloudflared 2025.8.1 → 2026.11.0 · bore 0.6.0".
func providerSummary(report []process.ProviderBinary) string {
	parts := make([]string, 0, len(report))
	for _, b := range report {
		if !b.Installed || b.Source == process.SourceSystem {
			continue
		}

		part := string(b.ID)
		if b.Version != "" {
			part += " " + b.Version
		}
		if b.HasUpdate && b.Latest != "" {
			part += " → " + b.Latest
		}
		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return i18n.T("providers_settings_none")
	}
	return strings.Join(parts, " · ")
}

func (m *Model) handleProviderSummary(msg providerSummaryMsg) (tea.Model, tea.Cmd) {
	if m.SettingsView != nil {
		m.SettingsView.Providers = msg.line
	}
	return m, nil
}
