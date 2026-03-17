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
	if m.Width == 0 || m.Height == 0 {
		return "Loading..."
	}

	if len(m.Items) == 0 {
		return m.viewEmptyState()
	}

	if m.Width >= TwoColumnThreshold {
		return m.viewTwoColumn()
	}

	return m.viewSingleColumn()
}

func (m *Model) viewTwoColumn() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold)).
		Bold(true)
	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim))

	title := headerStyle.Render("🎲  Foundry Tunnel Manager")
	version := versionStyle.Render("v" + version.Version)

	b.WriteString(title)
	b.WriteString(strings.Repeat(" ", m.Width-lipgloss.Width(title)-lipgloss.Width(version)-headerMargin))
	b.WriteString(version)
	b.WriteString("\n\n")

	// Calculate column widths
	leftWidth := int(float64(m.Width) * 0.4)
	rightWidth := m.Width - leftWidth - 3

	// Left column header
	leftHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorText)).
		Render("Your Connections")

	// Right column header
	rightHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorText)).
		Render("Selected Tunnel")

	b.WriteString(leftHeader)
	b.WriteString(strings.Repeat(" ", leftWidth-lipgloss.Width(leftHeader)+3))
	b.WriteString(rightHeader)
	b.WriteString("\n")

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBronze)).
		Render(strings.Repeat("─", m.Width-2))
	b.WriteString(divider)
	b.WriteString("\n")

	// Render list and detail side by side
	listContent := m.renderTunnelList(leftWidth - 2)
	detailContent := m.renderDetailPanel(rightWidth - 2)

	listLines := strings.Split(listContent, "\n")
	detailLines := strings.Split(detailContent, "\n")

	maxLines := len(listLines)
	if len(detailLines) > maxLines {
		maxLines = len(detailLines)
	}

	for i := 0; i < maxLines; i++ {
		leftLine := ""
		if i < len(listLines) {
			leftLine = listLines[i]
		}
		rightLine := ""
		if i < len(detailLines) {
			rightLine = detailLines[i]
		}

		// Pad left line to column width
		leftLine = lipgloss.NewStyle().Width(leftWidth).Render(leftLine)

		b.WriteString(leftLine)
		b.WriteString(" │ ")
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	// Help bar
	b.WriteString("\n")
	b.WriteString(m.renderHelpBar())

	return b.String()
}

func (m *Model) viewSingleColumn() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold)).
		Bold(true)
	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim))

	title := headerStyle.Render("🎲  Foundry Tunnel Manager")
	version := versionStyle.Render("v" + version.Version)

	b.WriteString(title)
	b.WriteString(strings.Repeat(" ", m.Width-lipgloss.Width(title)-lipgloss.Width(version)-headerMargin))
	b.WriteString(version)
	b.WriteString("\n\n")

	if m.App.WebServer != nil {
		urlStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGold))
		b.WriteString(urlStyle.Render("🌐  Dashboard: " + m.App.WebServer.URL() + " (press 'w')"))
		b.WriteString("\n\n")
	}

	b.WriteString(m.renderTunnelList(m.Width - 2))
	b.WriteString("\n")
	b.WriteString(m.renderHelpBar())

	return b.String()
}

func (m *Model) renderTunnelList(width int) string {
	var b strings.Builder

	for i, item := range m.Items {
		tunnelItem := item.(TunnelItem)
		line := m.renderTunnelListItem(i, tunnelItem, width)
		b.WriteString(line)
		if i < len(m.Items)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m *Model) renderTunnelListItem(idx int, item TunnelItem, width int) string {
	selected := idx == m.Cursor

	var statusEmoji, statusColor, bgColor string
	if item.Status.Running {
		if item.Status.Starting {
			statusEmoji = "🟡"
			statusColor = ColorText
			bgColor = ColorStarting
		} else {
			statusEmoji = "🟢"
			statusColor = ColorText
			bgColor = ColorOnline
		}
	} else {
		statusEmoji = "🔴"
		statusColor = ColorTextDim
		bgColor = ColorOffline
	}

	// Build content
	var parts []string

	if selected {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGold)).Render("▶"))
	} else {
		parts = append(parts, " ")
	}

	parts = append(parts, statusEmoji)

	nameStyle := lipgloss.NewStyle().
		Bold(selected).
		Foreground(lipgloss.Color(ColorText))
	parts = append(parts, nameStyle.Render(truncate(item.Tunnel.Name, 20)))

	// Status text
	var statusText string
	if item.Status.Running {
		if item.Status.Starting {
			statusText = "Starting..."
		} else {
			statusText = "Online"
		}
	} else {
		statusText = "Offline"
	}
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor))

	parts = append(parts, statusStyle.Render(statusText))

	// Provider and port
	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim))
	meta := fmt.Sprintf("%s • :%d", item.Tunnel.Provider, item.Tunnel.LocalPort)
	parts = append(parts, metaStyle.Render(meta))

	content := strings.Join(parts, "  ")

	// Apply background
	itemStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(bgColor)).
		Width(width).
		Padding(0, 1)

	if selected {
		itemStyle = itemStyle.BorderStyle(lipgloss.Border{
			Left: "█",
		}).BorderLeft(true).BorderForeground(lipgloss.Color(ColorGold))
	}

	return itemStyle.Render(content)
}

func (m *Model) renderDetailPanel(width int) string {
	var b strings.Builder

	item, ok := m.selectedItem()
	if !ok {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextDim)).
			Render("Select a tunnel to view details")
	}

	tunnel := item.Tunnel

	// Name
	nameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorGold)).
		Width(width)
	b.WriteString(nameStyle.Render(tunnel.Name))
	b.WriteString("\n\n")

	// Provider
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Provider:"))
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)).Render(string(tunnel.Provider)))
	b.WriteString("\n")

	// Local Port
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Local Port:"))
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)).Render(fmt.Sprintf(":%d", tunnel.LocalPort)))
	b.WriteString("\n\n")

	// Status
	var statusEmoji, statusText string
	if item.Status.Running {
		if item.Status.Starting {
			statusEmoji = "🟡"
			statusText = "STARTING"
		} else {
			statusEmoji = "🟢"
			statusText = "ONLINE"
		}
	} else {
		statusEmoji = "🔴"
		statusText = "OFFLINE"
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Status:"))
	b.WriteString(" ")
	b.WriteString(fmt.Sprintf("%s %s", statusEmoji, statusText))
	b.WriteString("\n\n")

	// Public URL
	if item.Status.Running && !item.Status.Starting && item.Status.PublicURL != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Public URL:"))
		b.WriteString("\n")

		urlBox := URLBoxStyle.Width(width - 2).Render(item.Status.PublicURL)
		b.WriteString(urlBox)
		b.WriteString("\n")

		copyHint := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBronze)).
			Render("Press 'c' to copy")
		b.WriteString(copyHint)
		b.WriteString("\n\n")
	}

	// Action buttons
	var actions []string
	if item.Status.Running {
		actions = append(actions, ButtonStyle.Render("[s] Stop"))
	} else {
		actions = append(actions, ButtonStyle.Render("[s] Start"))
	}
	actions = append(actions, ButtonStyle.Render("[l] Logs"))
	actions = append(actions, ButtonStyle.Render("[d] Delete"))

	b.WriteString(strings.Join(actions, "  "))

	return b.String()
}

func (m *Model) renderHelpBar() string {
	return ""
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
