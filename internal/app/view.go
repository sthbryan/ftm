package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"foundry-tunnel/internal/version"
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
	case viewDownloading:
		return m.viewDownloading()
	default:
		return m.viewList()
	}
}

const headerMargin = 4

func center(s string, width int) string {
	pad := (width - lipgloss.Width(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

func (m *Model) viewEmptyState() string {
	var b strings.Builder

	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold)).
		Render("🌐  +  🎲")

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold)).
		Bold(true).
		Render("Welcome, Dungeon Master!")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorText)).
		Render("You haven't created any tunnels yet.")

	desc := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim)).
		Render("Tunnels let your players connect to your Foundry world.")

	cta := ButtonActiveStyle.Render("[ Create First Tunnel ]")

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim)).
		Render("Or press 'a' to start")

	tip := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBronze)).
		Render("💡 Tip: You can also use the web dashboard at " + m.App.WebServer.URL())

	// Center everything vertically
	contentHeight := 12
	paddingTop := (m.Height - contentHeight) / 2
	if paddingTop < 2 {
		paddingTop = 2
	}

	b.WriteString(strings.Repeat("\n", paddingTop))
	b.WriteString(center(icon, m.Width) + "\n\n")
	b.WriteString(center(title, m.Width) + "\n\n")
	b.WriteString(center(subtitle, m.Width) + "\n")
	b.WriteString(center(desc, m.Width) + "\n\n")
	b.WriteString(center(cta, m.Width) + "\n\n")
	b.WriteString(center(hint, m.Width) + "\n\n")
	b.WriteString(center(tip, m.Width) + "\n")

	return b.String()
}

func (m *Model) viewList() string {
	var b strings.Builder

	// Header with version
	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim))

	title := TitleAccentStyle.Render("🎲  Foundry Tunnel Manager")
	version := versionStyle.Render("v" + version.Version)

	b.WriteString(title)
	spacing := m.Width - lipgloss.Width(title) - lipgloss.Width(version) - headerMargin
	if spacing < 1 {
		spacing = 1
	}
	b.WriteString(strings.Repeat(" ", spacing))
	b.WriteString(version)
	b.WriteString("\n\n")

	if m.App.WebServer != nil {
		urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#1E90FF"))
		b.WriteString(urlStyle.Render(fmt.Sprintf(" 🌐  Dashboard: %s (press 'w' to open)", m.App.WebServer.URL())))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.Items) == 0 {
		return m.viewEmptyState()
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
			parts = append(parts, URLStyle.Render("│ "+truncate(item.Status.PublicURL, 40)))
		}
	}

	return strings.Join(parts, " ")
}

func (m *Model) viewLogs() string {
	var b strings.Builder

	title := TitleStyle.Render(" 📋  TUNNEL LOGS  ")
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

	title := TitleStyle.Render(" ➕  ADD NEW TUNNEL  ")
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
	if m.FormFocus == 3 {
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

func (m *Model) viewDownloading() string {
	var b strings.Builder

	title := TitleStyle.Render(" ⬇️  INSTALLING  ")
	b.WriteString(title)
	b.WriteString("\n\n")

	percent := m.DownloadProgress.Percent

	var step string
	switch {
	case percent < 45:
		step = "Downloading Node.js..."
	case percent < 50:
		step = "Extracting..."
	case percent < 100:
		step = "Installing tunnelmole via npm..."
	default:
		step = "Complete!"
	}

	b.WriteString(fmt.Sprintf("%s\n\n", step))

	barWidth := 40
	filled := int(float64(barWidth) * percent / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	b.WriteString(fmt.Sprintf("[%s] %d%%\n", bar, int(percent)))

	if m.DownloadProgress.Total > 0 && percent < 50 {
		b.WriteString(fmt.Sprintf("%.1f MB / %.1f MB\n",
			float64(m.DownloadProgress.Current)/(1024*1024),
			float64(m.DownloadProgress.Total)/(1024*1024)))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("esc: cancel"))

	return b.String()
}
