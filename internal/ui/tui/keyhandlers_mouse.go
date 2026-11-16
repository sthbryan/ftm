package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/ui/tui/views"
)

func (m *Model) itemAt(row int) (int, bool) {
	if views.ItemHeight <= 0 {
		return 0, false
	}

	offset := row - m.listTop
	if offset < 0 {
		return 0, false
	}

	index := m.listFirst + offset/views.ItemHeight
	if index >= len(m.Items) {
		return 0, false
	}

	return index, true
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.State != viewList {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		if index, ok := m.itemAt(msg.Mouse().Y); ok {
			m.Cursor = index
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.moveCursorUp()
		case tea.MouseWheelDown:
			m.moveCursorDown()
		}
	}

	return m, nil
}
