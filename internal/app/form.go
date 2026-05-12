package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
)

func (m *Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.FormFocus = (m.FormFocus + 1) % 5

	case "shift+tab":
		m.FormFocus = (m.FormFocus - 1 + 5) % 5

	case "enter":
		return m.handleFormEnter()

	case "esc":
		m.State = viewList
		m.editingTunnelID = ""

	case "left", "right":
		m.handleProviderNav(msg.String())

	default:
		m.handleFormInput(msg.String())
	}

	return m, nil
}

func (m *Model) handleFormEnter() (tea.Model, tea.Cmd) {
	switch m.FormFocus {
	case 4:
		m.submitForm()
	default:
		m.FormFocus++
	}
	return m, nil
}

func (m *Model) handleProviderNav(dir string) {
	if m.FormFocus != 1 {
		return
	}

	providers := []config.Provider{
		config.ProviderCloudflared,
		config.ProviderTunnelmole,
		config.ProviderLocalhostRun,
		config.ProviderServeo,
		config.ProviderPinggy,
		config.ProviderBore,
	}

	current := config.Provider(m.FormValues.Provider)
	idx := -1
	for i, p := range providers {
		if p == current {
			idx = i
			break
		}
	}

	if idx == -1 {
		m.FormValues.Provider = string(config.ProviderCloudflared)
		return
	}

	if dir == "right" {
		idx = (idx + 1) % len(providers)
	} else {
		idx = (idx - 1 + len(providers)) % len(providers)
	}

	m.FormValues.Provider = string(providers[idx])
}

func (m *Model) handleFormInput(s string) {
	switch m.FormFocus {
	case 0:
		m.handleNameInput(s)

	case 2:
		m.handlePortInput(s)
	}
}

func (m *Model) handleNameInput(s string) {
	if s == "backspace" {
		if len(m.FormValues.Name) > 0 {
			m.FormValues.Name = m.FormValues.Name[:len(m.FormValues.Name)-1]
		}
	} else if len(s) == 1 {
		m.FormValues.Name += s
	}
}

func (m *Model) handlePortInput(s string) {
	if s == "backspace" {
		if len(m.FormValues.Port) > 0 {
			m.FormValues.Port = m.FormValues.Port[:len(m.FormValues.Port)-1]
		}
	} else if s >= "0" && s <= "9" && len(m.FormValues.Port) < 5 {
		m.FormValues.Port += s
	}
}

func (m *Model) submitForm() {
	if m.FormValues.Name == "" || m.FormValues.Port == "" {
		m.showMessage(i18n.T("validation_required_fields"))
		return
	}

	if m.editingTunnelID != "" {
		m.submitEditForm()
	} else {
		m.submitAddForm()
	}
}

func (m *Model) submitEditForm() {
	for i := range m.App.Config.Tunnels {
		if m.App.Config.Tunnels[i].ID == m.editingTunnelID {
			m.App.Config.Tunnels[i].Name = m.FormValues.Name
			m.App.Config.Tunnels[i].Provider = config.Provider(m.FormValues.Provider)
			m.App.Config.Tunnels[i].LocalPort = parsePort(m.FormValues.Port)

			if m.App.WebServer != nil {
				m.App.WebServer.BroadcastTunnelUpdate(m.App.Config.Tunnels[i])
			}
			break
		}
	}
	m.editingTunnelID = ""
	m.App.SaveConfig()
	m.refreshItems()
	m.State = viewList
	m.showMessage(i18n.T("tunnel_updated"))
}

func (m *Model) submitAddForm() {
	id := strings.ToLower(strings.ReplaceAll(m.FormValues.Name, " ", "-"))

	tunnel := config.TunnelConfig{
		ID:        id,
		Name:      m.FormValues.Name,
		Provider:  config.Provider(m.FormValues.Provider),
		LocalPort: parsePort(m.FormValues.Port),
	}

	m.App.Config.AddTunnel(tunnel)
	m.App.SaveConfig()
	m.refreshItems()
	m.State = viewList
	m.showMessage(i18n.T("tunnel_added"))
}

func parsePort(s string) int {
	var port int
	for _, c := range s {
		port = port*10 + int(c-'0')
	}
	return port
}
