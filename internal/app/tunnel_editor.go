package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
)

const (
	editorStops  = 4
	editorSubmit = editorStops - 1
)

func (m *Model) handleEditorKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		m.moveEditorFocus(1)

	case "shift+tab", "up":
		m.moveEditorFocus(-1)

	case "enter":
		return m.handleEditorEnter()

	case "esc":
		m.State = viewList
		m.editingTunnelID = ""

	case "left", "right":
		m.handleProviderNav(msg.String())

	default:
		m.handleEditorInput(msg.String())
	}

	return m, nil
}

func (m *Model) moveEditorFocus(delta int) {
	m.EditorFocus = (m.EditorFocus + delta + editorStops) % editorStops
}

func (m *Model) handleEditorEnter() (tea.Model, tea.Cmd) {
	if m.EditorFocus == editorSubmit {
		m.saveTunnel()
		return m, nil
	}

	m.moveEditorFocus(1)
	return m, nil
}

func (m *Model) handleProviderNav(dir string) {
	if m.EditorFocus != 1 {
		return
	}

	providers := config.AllProviders()

	idx := 0
	for i, p := range providers {
		if p == config.Provider(m.Draft.Provider) {
			idx = i
			break
		}
	}

	if dir == "right" {
		idx = (idx + 1) % len(providers)
	} else {
		idx = (idx - 1 + len(providers)) % len(providers)
	}

	m.Draft.Provider = string(providers[idx])
}

func (m *Model) handleEditorInput(s string) {
	switch m.EditorFocus {
	case 0:
		m.handleNameInput(s)

	case 2:
		m.handlePortInput(s)
	}
}

func (m *Model) handleNameInput(s string) {
	if s == "backspace" {
		if len(m.Draft.Name) > 0 {
			m.Draft.Name = m.Draft.Name[:len(m.Draft.Name)-1]
		}
	} else if s == "space" {
		m.Draft.Name += " "
	} else if len(s) == 1 {
		m.Draft.Name += s
	}
}

func (m *Model) handlePortInput(s string) {
	if s == "backspace" {
		if len(m.Draft.Port) > 0 {
			m.Draft.Port = m.Draft.Port[:len(m.Draft.Port)-1]
		}
	} else if s >= "0" && s <= "9" && len(m.Draft.Port) < 5 {
		m.Draft.Port += s
	}
}

func (m *Model) saveTunnel() {
	if m.Draft.Name == "" || m.Draft.Port == "" {
		m.showMessage(i18n.T("validation_required_fields"))
		return
	}

	if m.editingTunnelID != "" {
		m.updateTunnel()
	} else {
		m.createTunnel()
	}
}

func (m *Model) updateTunnel() {
	var updated *config.TunnelConfig
	saveErr := m.App.UpdateConfig(func(c *config.Config) error {
		t := c.GetTunnel(m.editingTunnelID)
		if t == nil {
			return nil
		}
		t.Name = m.Draft.Name
		t.Provider = config.Provider(m.Draft.Provider)
		t.LocalPort = parsePort(m.Draft.Port)
		copied := *t
		updated = &copied
		return nil
	})
	if saveErr == nil && updated != nil && m.App.WebServer != nil {
		m.App.WebServer.BroadcastTunnelUpdate(*updated)
	}
	m.editingTunnelID = ""
	m.refreshItems()
	m.State = viewList

	if saveErr != nil {
		m.showMessage(i18n.T("error_saving_config") + ": " + saveErr.Error())
	} else {
		m.showMessage(i18n.T("tunnel_updated"))
	}
}

func (m *Model) createTunnel() {
	var tunnel config.TunnelConfig
	saveErr := m.App.UpdateConfig(func(c *config.Config) error {
		tunnel = config.TunnelConfig{
			ID:        c.NewTunnelID(m.Draft.Name),
			Name:      m.Draft.Name,
			Provider:  config.Provider(m.Draft.Provider),
			LocalPort: parsePort(m.Draft.Port),
		}
		c.AddTunnel(tunnel)
		return nil
	})
	m.refreshItems()
	m.State = viewList

	if saveErr != nil {
		m.showMessage(i18n.T("error_saving_config") + ": " + saveErr.Error())
	} else {
		m.showMessage(i18n.T("tunnel_added"))
	}
}

func parsePort(s string) int {
	var port int
	for _, c := range s {
		port = port*10 + int(c-'0')
	}
	return port
}
