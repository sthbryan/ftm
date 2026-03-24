package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/version"
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

	leftWidth := int(float64(m.Width) * 0.4)
	rightWidth := m.Width - leftWidth - 3

	leftHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorText)).
		Render("Your Connections")

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

		leftLine = lipgloss.NewStyle().Width(leftWidth).Render(leftLine)

		b.WriteString(leftLine)
		b.WriteString(" │ ")
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.Message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGold)).
			Bold(true)
		b.WriteString(msgStyle.Render("✓ " + m.Message))
		b.WriteString("\n")
	}

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

	if m.Message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGold)).
			Bold(true)
		b.WriteString(msgStyle.Render("✓ " + m.Message))
		b.WriteString("\n")
	}

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
	switch item.Status.State {
	case config.TunnelStateStarting, config.TunnelStateConnecting:
		statusEmoji = "🟡"
		statusColor = ColorText
		bgColor = ColorConnecting
	case config.TunnelStateOnline:
		statusEmoji = "🟢"
		statusColor = ColorText
		bgColor = ColorOnline
	case config.TunnelStateError, config.TunnelStateTimeout:
		statusEmoji = "🔴"
		statusColor = ColorText
		bgColor = ColorError
	case config.TunnelStateStopped:
		statusEmoji = "⚪"
		statusColor = ColorTextDim
		bgColor = ColorStopped
	default:
		statusEmoji = "⚪"
		statusColor = ColorTextDim
		bgColor = ColorOffline
	}

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

	var statusText string
	switch item.Status.State {
	case config.TunnelStateStarting:
		statusText = "Starting..."
	case config.TunnelStateConnecting:
		statusText = "Connecting..."
	case config.TunnelStateOnline:
		statusText = "Online"
	case config.TunnelStateError:
		statusText = "Error"
	case config.TunnelStateTimeout:
		statusText = "Timeout"
	case config.TunnelStateStopped:
		statusText = "Offline"
	default:
		statusText = "Offline"
	}
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor))

	parts = append(parts, statusStyle.Render(statusText))

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim))
	meta := fmt.Sprintf("%s • :%d", item.Tunnel.Provider, item.Tunnel.LocalPort)
	parts = append(parts, metaStyle.Render(meta))

	content := strings.Join(parts, "  ")

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

	nameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorGold)).
		Width(width)
	b.WriteString(nameStyle.Render(tunnel.Name))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Provider:"))
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)).Render(string(tunnel.Provider)))
	b.WriteString("\n")

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Local Port:"))
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)).Render(fmt.Sprintf(":%d", tunnel.LocalPort)))
	b.WriteString("\n\n")

	var statusEmoji, statusText string
	switch item.Status.State {
	case config.TunnelStateStarting:
		statusEmoji = "🟡"
		statusText = "STARTING"
	case config.TunnelStateConnecting:
		statusEmoji = "🟡"
		statusText = "CONNECTING"
	case config.TunnelStateOnline:
		statusEmoji = "🟢"
		statusText = "ONLINE"
	case config.TunnelStateError:
		statusEmoji = "🔴"
		statusText = "ERROR"
	case config.TunnelStateStopped:
		statusEmoji = "⚪"
		statusText = "OFFLINE"
	default:
		statusEmoji = "⚪"
		statusText = "OFFLINE"
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Status:"))
	b.WriteString(" ")
	b.WriteString(fmt.Sprintf("%s %s", statusEmoji, statusText))
	b.WriteString("\n\n")

	if item.Status.State == config.TunnelStateOnline && item.Status.PublicURL != "" {
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

	if item.Status.ErrorMessage != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextDim)).Render("Error:"))
		b.WriteString("\n")

		errorBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Width(width - 2).Render(item.Status.ErrorMessage)
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	var actions []string
	isActive := item.Status.State == config.TunnelStateOnline || item.Status.State == config.TunnelStateStarting || item.Status.State == config.TunnelStateConnecting
	if isActive {
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
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim)).
		Background(lipgloss.Color(ColorBg))

	shortcuts := []string{
		"↑/↓ navigate",
		"Enter toggle",
		"a add",
		"l logs",
		"w web",
		"o open config",
		"q quit",
	}

	content := strings.Join(shortcuts, "  •  ")

	return helpStyle.Render(content)
}

func (m *Model) viewLogs() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold)).
		Bold(true).
		Render("📋  Tunnel Logs")
	b.WriteString(header)
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

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorText)).
		Bold(true)
	b.WriteString(nameStyle.Render(tunnelName))
	b.WriteString("\n")

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBronze)).
		Render(strings.Repeat("─", m.Width-2))
	b.WriteString(divider)
	b.WriteString("\n")

	m.updateLogViewport()
	b.WriteString(m.LogViewport.View())

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim)).
		Render("esc/b: back • ↑/↓: scroll"))

	return b.String()
}

func (m *Model) viewAddForm() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold)).
		Bold(true).
		Render("➕  Add New Tunnel")
	b.WriteString(header)
	b.WriteString("\n\n")

	inputWidth := 25

	nameLabel := "Name"
	nameValue := m.FormValues.Name
	if nameValue == "" {
		nameValue = ""
	}
	nameHint := ""
	if m.FormFocus == 0 {
		nameLabel = "▶ Name"
		nameHint = "type to enter"
	}

	labelStyle := lipgloss.NewStyle().Width(15).Foreground(lipgloss.Color(ColorTextDim))
	if m.FormFocus == 0 {
		labelStyle = labelStyle.Bold(true).Foreground(lipgloss.Color(ColorGold))
	}

	nameStyle := lipgloss.NewStyle().
		Width(inputWidth).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder())
	if m.FormFocus == 0 {
		nameStyle = nameStyle.
			BorderForeground(lipgloss.Color(ColorGold)).
			Foreground(lipgloss.Color(ColorText))
	} else {
		nameStyle = nameStyle.
			BorderForeground(lipgloss.Color(ColorBronze)).
			Foreground(lipgloss.Color(ColorTextDim))
	}

	b.WriteString(labelStyle.Render(nameLabel + ":"))
	b.WriteString("\n")
	b.WriteString(nameStyle.Render(nameValue))
	if nameHint != "" {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBronze))
		b.WriteString(" " + hintStyle.Render(nameHint))
	}
	b.WriteString("\n")
	b.WriteString("\n")

	providerLabel := "Provider"
	providerValue := m.FormValues.Provider
	providerHint := ""
	if m.FormFocus == 1 {
		providerLabel = "▶ Provider"
		providerValue = "← " + providerValue + " →"
		providerHint = "arrows to change"
	}

	labelStyle = lipgloss.NewStyle().Width(15).Foreground(lipgloss.Color(ColorTextDim))
	if m.FormFocus == 1 {
		labelStyle = labelStyle.Bold(true).Foreground(lipgloss.Color(ColorGold))
	}

	providerStyle := lipgloss.NewStyle().
		Width(inputWidth).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder())
	if m.FormFocus == 1 {
		providerStyle = providerStyle.
			BorderForeground(lipgloss.Color(ColorGold)).
			Foreground(lipgloss.Color(ColorText))
	} else {
		providerStyle = providerStyle.
			BorderForeground(lipgloss.Color(ColorBronze)).
			Foreground(lipgloss.Color(ColorTextDim))
	}

	b.WriteString(labelStyle.Render(providerLabel + ":"))
	b.WriteString("\n")
	b.WriteString(providerStyle.Render(providerValue))
	if providerHint != "" {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBronze))
		b.WriteString(" " + hintStyle.Render(providerHint))
	}
	b.WriteString("\n")
	b.WriteString("\n")

	portLabel := "Local Port"
	portValue := m.FormValues.Port
	if portValue == "" {
		portValue = ""
	}
	portHint := ""
	if m.FormFocus == 2 {
		portLabel = "▶ Local Port"
		portHint = "numbers only"
	}

	labelStyle = lipgloss.NewStyle().Width(15).Foreground(lipgloss.Color(ColorTextDim))
	if m.FormFocus == 2 {
		labelStyle = labelStyle.Bold(true).Foreground(lipgloss.Color(ColorGold))
	}

	portStyle := lipgloss.NewStyle().
		Width(inputWidth).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder())
	if m.FormFocus == 2 {
		portStyle = portStyle.
			BorderForeground(lipgloss.Color(ColorGold)).
			Foreground(lipgloss.Color(ColorText))
	} else {
		portStyle = portStyle.
			BorderForeground(lipgloss.Color(ColorBronze)).
			Foreground(lipgloss.Color(ColorTextDim))
	}

	b.WriteString(labelStyle.Render(portLabel + ":"))
	b.WriteString("\n")
	b.WriteString(portStyle.Render(portValue))
	if portHint != "" {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBronze))
		b.WriteString(" " + hintStyle.Render(portHint))
	}
	b.WriteString("\n")
	b.WriteString("\n")

	submitStyle := lipgloss.NewStyle()
	if m.FormFocus == 3 {
		submitStyle = ButtonActiveStyle
	} else {
		submitStyle = ButtonStyle
	}
	b.WriteString(lipgloss.NewStyle().Width(16).Render(""))
	b.WriteString(submitStyle.Render(" Submit "))

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim)).
		Render("tab: next • enter: submit • esc: cancel"))

	return b.String()
}

func (m *Model) viewDownloading() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold)).
		Bold(true).
		Render("⬇️  Installing")
	b.WriteString(header)
	b.WriteString("\n\n")

	percent := m.DownloadProgress.Percent
	name := m.DownloadProgress.Name
	if name == "" {
		name = "binary"
	}

	var step string
	switch {
	case percent < 90:
		step = fmt.Sprintf("Downloading %s...", name)
	case percent < 100:
		step = fmt.Sprintf("Installing %s...", name)
	default:
		step = "Complete!"
	}

	stepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorText))
	b.WriteString(stepStyle.Render(step))
	b.WriteString("\n\n")

	barWidth := 40
	filled := int(float64(barWidth) * percent / 100)

	barStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGold))
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBronze))

	bar := barStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", barWidth-filled))

	b.WriteString(fmt.Sprintf("[%s] %d%%\n", bar, int(percent)))

	if m.DownloadProgress.Total > 0 && percent < 50 {
		sizeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextDim))
		b.WriteString(sizeStyle.Render(fmt.Sprintf("%.1f MB / %.1f MB",
			float64(m.DownloadProgress.Current)/(1024*1024),
			float64(m.DownloadProgress.Total)/(1024*1024))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorTextDim)).
		Render("esc: cancel"))

	return b.String()
}
