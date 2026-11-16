package main

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type refresh struct{}
type tick struct{}

// view is the terminal: the world, who is at the table, the link players go
// through, and the keys.
type view struct {
	t       *table
	link    string
	input   textinput.Model
	editing bool
}

func newView(t *table, link string) *view {
	input := textinput.New()
	input.Placeholder = "https://….trycloudflare.com"
	input.SetWidth(60)
	return &view{t: t, link: link, input: input}
}

func (v *view) Init() tea.Cmd { return every() }

func every() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tick{} })
}

func (v *view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// While the link is being edited the input gets everything, pastes
	// included: a terminal paste arrives as tea.PasteMsg, not as keys.
	if v.editing {
		if key, ok := msg.(tea.KeyPressMsg); ok {
			switch key.String() {
			case "enter":
				v.link = strings.TrimSpace(v.input.Value())
				v.editing = false
				v.input.Blur()
				return v, nil
			case "esc":
				v.editing = false
				v.input.Blur()
				return v, nil
			}
		}
		if _, ok := msg.(tick); ok {
			return v, every()
		}
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return v, tea.Quit
		case "+", "=", "p":
			v.t.addBot(v.link)
		case "-", "_":
			v.t.dropBot()
		case "l":
			v.editing = true
			v.input.SetValue(v.link)
			return v, v.input.Focus()
		case "s":
			v.t.mu.Lock()
			v.t.setup = !v.t.setup
			v.t.mu.Unlock()
		case "w":
			v.t.mu.Lock()
			v.t.world = (v.t.world + 1) % len(worlds)
			v.t.mu.Unlock()
		}
	case tick:
		return v, every()
	}
	return v, nil
}

var (
	gold  = lipgloss.NewStyle().Foreground(lipgloss.Color("#d4a84a")).Bold(true)
	muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a8494"))
	ok    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6fd3a0"))
	warn  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0bc5e"))
	key   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ece8f4")).Bold(true)
)

func (v *view) View() tea.View {
	t := v.t
	t.mu.Lock()
	defer t.mu.Unlock()

	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s %s\n\n", gold.Render("fake Foundry"), muted.Render(fmt.Sprintf("v%s · localhost:%d", t.version, t.port)))

	if t.setup {
		fmt.Fprintf(&b, "  %s no world launched %s\n", warn.Render("●"), muted.Render("(setup screen)"))
	} else {
		w := worlds[t.world]
		fmt.Fprintf(&b, "  %s %s %s\n", ok.Render("●"), w.id, muted.Render("("+w.system+")"))
		fmt.Fprintf(&b, "    %d %s + you (GM)  %s\n", len(t.players), plural(len(t.players), "player", "players"),
			muted.Render(fmt.Sprintf("· Foundry reports users: %d", len(t.players)+1)))
	}
	for id := 1; id <= t.next; id++ {
		p, here := t.players[id]
		if !here {
			continue
		}
		via := muted.Render("direct, ftm doesn't see it")
		if p.tunnel {
			via = ok.Render("through the tunnel")
		}
		fmt.Fprintf(&b, "    · player %d  %s  %s\n", p.id, p.from, via)
	}

	b.WriteString("\n")
	if v.editing {
		fmt.Fprintf(&b, "  Link  %s\n  %s\n", v.input.View(), muted.Render("enter to keep · esc to cancel"))
	} else if v.link != "" {
		fmt.Fprintf(&b, "  Link  %s\n", v.link)
	} else {
		fmt.Fprintf(&b, "  Link  %s\n", muted.Render("none: + joins players directly, so ftm won't count them. Press l to paste the table's link"))
	}

	for _, line := range t.log {
		fmt.Fprintf(&b, "  %s\n", muted.Render(line))
	}

	fmt.Fprintf(&b, "\n  %s add player  %s remove  %s paste link  %s world  %s setup  %s quit\n",
		key.Render("+"), key.Render("-"), key.Render("l"), key.Render("w"), key.Render("s"), key.Render("q"))
	return tea.NewView(b.String())
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
