package tui

import (
	"charm.land/bubbles/v2/key"

	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/components"
)

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Toggle   key.Binding
	Logs     key.Binding
	Copy     key.Binding
	Web      key.Binding
	Add      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Config   key.Binding
	Settings key.Binding
	Update   key.Binding
	Back     key.Binding
	Help     key.Binding
	Quit     key.Binding

	ForceQuit key.Binding
}

var DefaultKeys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "previous"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "start/stop"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy URL"),
	),
	Web: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "open web"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Config: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open config"),
	),
	Settings: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "settings"),
	),
	Update: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "update"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	ForceQuit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Toggle},
		{k.Logs, k.Copy, k.Web},
		{k.Add, k.Edit, k.Delete},
		{k.Settings, k.Config, k.Update},
		{k.Back, k.Help, k.Quit},
	}
}

func (k KeyMap) listShortcuts() []components.Shortcut {
	return []components.Shortcut{
		{Keys: "↑↓/kj", Label: i18n.T("navigate")},
		{Keys: k.Enter.Help().Key + "/" + k.Toggle.Help().Key, Label: i18n.T("start_stop")},
		{Keys: k.Logs.Help().Key, Label: i18n.T("logs")},
		{Keys: k.Copy.Help().Key, Label: i18n.T("copy_url")},
		{Keys: k.Add.Help().Key, Label: i18n.T("create")},
		{Keys: k.Edit.Help().Key, Label: i18n.T("edit")},
		{Keys: k.Delete.Help().Key, Label: i18n.T("delete")},
		{Keys: k.Settings.Help().Key, Label: i18n.T("settings")},
		{Keys: k.Web.Help().Key, Label: i18n.T("web")},
		{Keys: k.Config.Help().Key, Label: i18n.T("config")},
		{Keys: k.Quit.Help().Key, Label: i18n.T("close")},
	}
}
