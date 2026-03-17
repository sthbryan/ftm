package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Loading..."
	}
	
	switch m.State {
	case viewList:
		return m.viewList()
	case viewLogs:
		return m.viewLogs()
	case viewAddForm:
		return m.viewAddForm()
	default:
		return m.viewList()
	}
}

func (m *Model) viewList() string {
	var b strings.Builder
	
	title := TitleStyle.Render(" 🎲 Foundry Tunnel Manager ")
	b.WriteString(title)
	b.WriteString("\n\n")
	
	if len(m.Items) == 0 {
		b.WriteString("No tunnels configured. Press 'a' to add one.\n")
	} else {
		for i, item := range m.Items {
			tunnelItem := item.(TunnelItem)
			b.WriteString(m.renderTunnelItem(i, tunnelItem))
			b.WriteString("\n")
		}
	}
	
	b.WriteString("\n")
	
	if m.Message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)
		b.WriteString(msgStyle.Render(m.Message))
		b.WriteString("\n")
	}
	
	b.WriteString("\n")
	b.WriteString(m.Help.View(m.Keys))
	
	return b.String()
}

func (m *Model) renderTunnelItem(idx int, item TunnelItem) string {
	selected := idx == m.Cursor
	
	var statusStr, statusColor string
	if item.Status.Running {
		if item.Status.Starting {
			statusStr = "⟳ STARTING"
			statusColor = "#FFA500"
		} else {
			statusStr = "▶ ONLINE"
			statusColor = "#00FF00"
		}
	} else {
		statusStr = "● OFFLINE"
		statusColor = "#FF0000"
	}
	
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))
	
	var parts []string
	
	if selected {
		parts = append(parts, "▶ ")
	} else {
		parts = append(parts, "  ")
	}
	
	nameStyle := lipgloss.NewStyle().Bold(selected)
	if selected {
		nameStyle = nameStyle.Background(lipgloss.Color("#333333"))
	}
	parts = append(parts, nameStyle.Render(truncate(item.Tunnel.Name, 25)))
	parts = append(parts, fmt.Sprintf("│ %s", item.Tunnel.Provider))
	parts = append(parts, fmt.Sprintf("│ :%d", item.Tunnel.LocalPort))
	parts = append(parts, statusStyle.Render("│ "+statusStr))
	
	if item.Status.Running && !item.Status.Starting {
		if item.Status.PublicURL != "" {
			parts = append(parts, URLStyle.Render("│ "+truncate(item.Status.PublicURL, 30)))
		}
		
		if item.Tunnel.ShortURL != "" {
			shortDisplay := item.Tunnel.ShortURL
			if mapping, ok := m.App.URLCache.Get(item.Tunnel.ID); ok {
				shortDisplay = mapping.ShortURL
			}
			parts = append(parts, ShortURLStyle.Render("│ "+truncate(shortDisplay, 25)))
		}
	}
	
	return strings.Join(parts, " ")
}

func (m *Model) viewLogs() string {
	var b strings.Builder
	
	title := TitleStyle.Render(" 📋 Tunnel Logs ")
	b.WriteString(title)
	b.WriteString("\n\n")
	
	var tunnelName string
	if item, ok := m.selectedItem(); ok && item.Tunnel.ID == m.SelectedTunnel {
		tunnelName = item.Tunnel.Name
	} else {
		for _, t := range m.App.Config.Tunnels {
			if t.ID == m.SelectedTunnel {
				tunnelName = t.Name
				break
			}
		}
	}
	
	b.WriteString(fmt.Sprintf("Tunnel: %s", tunnelName))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).
		Render("─────────────────────────────────────────"))
	b.WriteString("\n")
	
	m.updateLogViewport()
	b.WriteString(m.LogViewport.View())
	
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("esc/b: back • ↑/↓: scroll"))
	
	return b.String()
}

func (m *Model) viewAddForm() string {
	var b strings.Builder
	
	title := TitleStyle.Render(" ➕ Add New Tunnel ")
	b.WriteString(title)
	b.WriteString("\n\n")
	
	fields := []struct {
		label   string
		value   string
		focused bool
	}{
		{"Name", m.FormValues.Name, m.FormFocus == 0},
		{"Provider", m.FormValues.Provider, m.FormFocus == 1},
		{"Local Port", m.FormValues.Port, m.FormFocus == 2},
		{"Short URL (optional)", m.FormValues.ShortURL, m.FormFocus == 3},
	}
	
	for _, f := range fields {
		labelStyle := lipgloss.NewStyle().Width(20)
		if f.focused {
			labelStyle = labelStyle.Bold(true).Foreground(lipgloss.Color("#FFD700"))
		}
		
		valueStyle := lipgloss.NewStyle()
		if f.focused {
			valueStyle = valueStyle.Background(lipgloss.Color("#333333"))
		}
		if f.value == "" {
			valueStyle = valueStyle.Foreground(lipgloss.Color("#666666"))
		}
		
		displayValue := f.value
		if displayValue == "" {
			displayValue = "..."
		}
		
		b.WriteString(labelStyle.Render(f.label + ":"))
		b.WriteString(" ")
		b.WriteString(valueStyle.Render(displayValue))
		b.WriteString("\n\n")
	}
	
	submitStyle := lipgloss.NewStyle()
	if m.FormFocus == 4 {
		submitStyle = submitStyle.Background(lipgloss.Color("#00AA00")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
	}
	b.WriteString(strings.Repeat(" ", 21))
	b.WriteString(submitStyle.Render(" [ Submit ] "))
	
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("tab: next • enter: submit • esc: cancel"))
	
	return b.String()
}
