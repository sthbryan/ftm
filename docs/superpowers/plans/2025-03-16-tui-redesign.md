# TUI Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the TUI with RPG Minimal theme: 2-column layout, status backgrounds, empty state, and contextual help.

**Architecture:** Split view rendering into separate components (list panel, detail panel, empty state). Use responsive layout that switches to single column for small terminals. All styling centralized in styles.go.

**Tech Stack:** Go, Bubble Tea (charmbracelet/bubbletea), Lipgloss

---

## File Structure

| File | Purpose |
|------|---------|
| `internal/app/styles.go` | All color definitions and reusable styles |
| `internal/app/view.go` | View functions for all screens |
| `internal/app/model.go` | Model state (minor additions for responsive layout) |

---

## Task 1: Define RPG Minimal Color Styles

**Files:**
- Modify: `internal/app/styles.go`

- [ ] **Step 1: Define color constants and base styles**

Add at the top of styles.go after imports:

```go
const (
    ColorBg         = "#1a1814"
    ColorGold       = "#c9a227"
    ColorBronze     = "#8b7355"
    ColorText       = "#e8e6e1"
    ColorTextDim    = "#9a9590"
    
    ColorOnline     = "#1e3a2f"
    ColorOffline    = "#2a2824"
    ColorStarting   = "#3a3020"
    ColorError      = "#3a2020"
)
```

Update TitleStyle:
```go
var TitleStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color(ColorGold))
```

- [ ] **Step 2: Define status background styles**

Add after TitleStyle:

```go
var StatusOnlineStyle = lipgloss.NewStyle().
    Background(lipgloss.Color(ColorOnline)).
    Foreground(lipgloss.Color(ColorText))

var StatusOfflineStyle = lipgloss.NewStyle().
    Background(lipgloss.Color(ColorOffline)).
    Foreground(lipgloss.Color(ColorText))

var StatusStartingStyle = lipgloss.NewStyle().
    Background(lipgloss.Color(ColorStarting)).
    Foreground(lipgloss.Color(ColorText))

var StatusErrorStyle = lipgloss.NewStyle().
    Background(lipgloss.Color(ColorError)).
    Foreground(lipgloss.Color(ColorText))
```

- [ ] **Step 3: Define panel and selection styles**

```go
var PanelStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color(ColorBronze)).
    Padding(1)

var SelectedStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color(ColorGold)).
    Bold(true)

var URLBoxStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color(ColorGold)).
    Background(lipgloss.Color(ColorBg)).
    Foreground(lipgloss.Color(ColorText)).
    Padding(0, 1)

var ButtonStyle = lipgloss.NewStyle().
    Background(lipgloss.Color(ColorBronze)).
    Foreground(lipgloss.Color(ColorBg)).
    Padding(0, 2)

var ButtonActiveStyle = lipgloss.NewStyle().
    Background(lipgloss.Color(ColorGold)).
    Foreground(lipgloss.Color(ColorBg)).
    Bold(true).
    Padding(0, 2)
```

- [ ] **Step 4: Commit**

```bash
git add internal/app/styles.go
git commit -m "style(tui): add rpg minimal color palette"
```

---

## Task 2: Implement Header Component

**Files:**
- Modify: `internal/app/view.go`

- [ ] **Step 1: Update viewList header**

Replace the header section in viewList():

```go
func (m *Model) viewList() string {
    var b strings.Builder
    
    // Header with version
    headerStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGold)).
        Bold(true)
    versionStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorTextDim))
    
    title := headerStyle.Render("🎲  Foundry Tunnel Manager")
    version := versionStyle.Render("v0.2.0")
    
    b.WriteString(title)
    b.WriteString(strings.Repeat(" ", m.Width-lipgloss.Width(title)-lipgloss.Width(version)-4))
    b.WriteString(version)
    b.WriteString("\n\n")
```

- [ ] **Step 2: Commit**

```bash
git add internal/app/view.go
git commit -m "feat(tui): redesign header with rpg theme"
```

---

## Task 3: Implement Empty State

**Files:**
- Modify: `internal/app/view.go`

- [ ] **Step 1: Create empty state view function**

Add new function before viewList:

```go
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
```

- [ ] **Step 2: Add center helper function**

```go
func center(s string, width int) string {
    pad := (width - lipgloss.Width(s)) / 2
    if pad < 0 {
        pad = 0
    }
    return strings.Repeat(" ", pad) + s
}
```

- [ ] **Step 3: Update viewList to use empty state**

Replace the empty check in viewList:

```go
if len(m.Items) == 0 {
    return m.viewEmptyState()
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/app/view.go
git commit -m "feat(tui): add empty state for first-time users"
```

---

## Task 4: Implement Responsive 2-Column Layout

**Files:**
- Modify: `internal/app/view.go`
- Modify: `internal/app/model.go`

- [ ] **Step 1: Add layout threshold constant**

In model.go, add to Model struct (if not exists):

```go
const TwoColumnThreshold = 100
```

- [ ] **Step 2: Create viewTwoColumn function**

Add to view.go:

```go
func (m *Model) viewTwoColumn() string {
    var b strings.Builder
    
    // Header
    headerStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGold)).
        Bold(true)
    versionStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorTextDim))
    
    title := headerStyle.Render("🎲  Foundry Tunnel Manager")
    version := versionStyle.Render("v0.2.0")
    
    b.WriteString(title)
    b.WriteString(strings.Repeat(" ", m.Width-lipgloss.Width(title)-lipgloss.Width(version)-4))
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
```

- [ ] **Step 3: Update viewList to use responsive layout**

Replace viewList logic:

```go
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
```

- [ ] **Step 4: Create viewSingleColumn function**

```go
func (m *Model) viewSingleColumn() string {
    var b strings.Builder
    
    headerStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGold)).
        Bold(true)
    versionStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorTextDim))
    
    title := headerStyle.Render("🎲  Foundry Tunnel Manager")
    version := versionStyle.Render("v0.2.0")
    
    b.WriteString(title)
    b.WriteString(strings.Repeat(" ", m.Width-lipgloss.Width(title)-lipgloss.Width(version)-4))
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
```

- [ ] **Step 5: Commit**

```bash
git add internal/app/view.go internal/app/model.go
git commit -m "feat(tui): implement responsive 2-column layout"
```

---

## Task 5: Implement Styled Tunnel List

**Files:**
- Modify: `internal/app/view.go`

- [ ] **Step 1: Create renderTunnelList function**

```go
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
```

- [ ] **Step 2: Create renderTunnelListItem function**

```go
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
```

- [ ] **Step 3: Commit**

```bash
git add internal/app/view.go
git commit -m "feat(tui): styled tunnel list with status backgrounds"
```

---

## Task 6: Implement Detail Panel

**Files:**
- Modify: `internal/app/view.go`

- [ ] **Step 1: Create renderDetailPanel function**

```go
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
    b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)).Render(tunnel.Provider))
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
```

- [ ] **Step 2: Commit**

```bash
git add internal/app/view.go
git commit -m "feat(tui): add detail panel with url box and actions"
```

---

## Task 7: Implement Contextual Help Bar

**Files:**
- Modify: `internal/app/view.go`

- [ ] **Step 1: Create renderHelpBar function**

```go
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
        "q quit",
    }
    
    content := strings.Join(shortcuts, "  •  ")
    
    return helpStyle.Render(content)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/app/view.go
git commit -m "feat(tui): add contextual help bar"
```

---

## Task 8: Add Copy URL Keybinding

**Files:**
- Modify: `internal/app/keys.go`
- Modify: `internal/app/update.go`

- [ ] **Step 1: Add 'c' key to keyMap**

In keys.go, add to keyMap struct:

```go
CopyURL key.Binding
```

In DefaultKeyMap(), add:

```go
CopyURL: key.NewBinding(
    key.WithKeys("c"),
    key.WithHelp("c", "copy url"),
),
```

- [ ] **Step 2: Handle 'c' key in update**

In update.go, add case in switch:

```go
case key.Matches(msg, m.Keys.CopyURL):
    if item, ok := m.selectedItem(); ok && item.Status.PublicURL != "" {
        if err := clipboard.WriteAll(item.Status.PublicURL); err == nil {
            m.Message = "URL copied to clipboard!"
        }
    }
```

- [ ] **Step 3: Commit**

```bash
git add internal/app/keys.go internal/app/update.go
git commit -m "feat(tui): add copy url keybinding"
```

---

## Task 9: Update Logs and Form Views

**Files:**
- Modify: `internal/app/view.go`

- [ ] **Step 1: Update viewLogs with RPG theme**

```go
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
```

- [ ] **Step 2: Update viewAddForm with RPG theme**

```go
func (m *Model) viewAddForm() string {
    var b strings.Builder
    
    header := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGold)).
        Bold(true).
        Render("➕  Add New Tunnel")
    b.WriteString(header)
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
        labelStyle := lipgloss.NewStyle().
            Width(15).
            Foreground(lipgloss.Color(ColorTextDim))
        if f.focused {
            labelStyle = labelStyle.Bold(true).Foreground(lipgloss.Color(ColorGold))
        }
        
        valueStyle := lipgloss.NewStyle().
            Foreground(lipgloss.Color(ColorText))
        if f.focused {
            valueStyle = valueStyle.Background(lipgloss.Color(ColorBg)).
                BorderStyle(lipgloss.NormalBorder()).
                BorderForeground(lipgloss.Color(ColorGold))
        }
        if f.value == "" {
            valueStyle = valueStyle.Foreground(lipgloss.Color(ColorTextDim))
        }
        
        displayValue := f.value
        if displayValue == "" {
            displayValue = "..."
        }
        
        b.WriteString(labelStyle.Render(f.label + ":"))
        b.WriteString(" ")
        b.WriteString(valueStyle.Render(" " + displayValue + " "))
        b.WriteString("\n\n")
    }
    
    submitStyle := lipgloss.NewStyle()
    if m.FormFocus == 3 {
        submitStyle = ButtonActiveStyle
    } else {
        submitStyle = ButtonStyle
    }
    b.WriteString(strings.Repeat(" ", 16))
    b.WriteString(submitStyle.Render(" Submit "))
    
    b.WriteString("\n\n")
    b.WriteString(lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorTextDim)).
        Render("tab: next • enter: submit • esc: cancel"))
    
    return b.String()
}
```

- [ ] **Step 3: Update viewDownloading with RPG theme**

```go
func (m *Model) viewDownloading() string {
    var b strings.Builder
    
    header := lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGold)).
        Bold(true).
        Render("⬇️  Installing")
    b.WriteString(header)
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
```

- [ ] **Step 4: Commit**

```bash
git add internal/app/view.go
git commit -m "style(tui): apply rpg theme to logs, form, and download views"
```

---

## Task 10: Final Integration and Test

**Files:**
- All modified files

- [ ] **Step 1: Build and run**

```bash
cd /Users/sthbryan/Documents/Codes/prtty
go build -o foundry-tunnel ./cmd/foundry-tunnel
./foundry-tunnel
```

- [ ] **Step 2: Verify in terminal**

Check:
- Header shows gold title with version aligned right
- Empty state shows centered welcome with CTA button
- With tunnels, 2-column layout appears (terminal ≥100 cols)
- Tunnel list has colored backgrounds for each status
- Selected tunnel has gold left border
- Detail panel shows URL in box with copy hint
- Help bar shows contextual shortcuts

- [ ] **Step 3: Final commit**

```bash
git add .
git commit -m "feat(tui): complete rpg minimal redesign"
```

---

## Testing Checklist

- [ ] Empty state displays correctly with no tunnels
- [ ] Adding first tunnel shows new layout
- [ ] 2-column layout works in large terminal (≥100 cols)
- [ ] Single-column layout works in small terminal (<100 cols)
- [ ] Each tunnel status has correct background color
- [ ] Selected tunnel has gold indicator
- [ ] Detail panel updates when selection changes
- [ ] URL box displays when tunnel is online
- [ ] Copy URL keybinding works
- [ ] Help bar shows all shortcuts
- [ ] Form view has RPG styling
- [ ] Logs view has RPG styling
- [ ] Download view has RPG styling
