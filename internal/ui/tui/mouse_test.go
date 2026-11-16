package tui

import (
	"testing"

	"charm.land/bubbles/v2/list"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/ui/tui/views"
)

func modelWithItems(count, listTop int) *Model {
	items := make([]list.Item, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, TunnelItem{
			Tunnel: core.TunnelConfig{ID: string(rune('a' + i))},
		})
	}

	return &Model{Items: items, listTop: listTop}
}

func TestItemAtMapsRowsToTunnels(t *testing.T) {
	const listTop = 4
	m := modelWithItems(3, listTop)

	for i := 0; i < 3; i++ {
		row := listTop + i*views.ItemHeight

		index, ok := m.itemAt(row)
		if !ok {
			t.Fatalf("row %d reported no tunnel, want index %d", row, i)
		}
		if index != i {
			t.Errorf("row %d selected tunnel %d, want %d", row, index, i)
		}
	}
}

func TestItemAtRejectsRowsOutsideTheList(t *testing.T) {
	const listTop = 4
	m := modelWithItems(3, listTop)

	rows := []int{0, 1, listTop - 1, listTop + 3*views.ItemHeight, 100}
	for _, row := range rows {
		if _, ok := m.itemAt(row); ok {
			t.Errorf("row %d reported a tunnel, want none", row)
		}
	}
}

func TestItemAtFollowsTheReportedListTop(t *testing.T) {
	for _, listTop := range []int{2, 4, 5, 6} {
		m := modelWithItems(2, listTop)

		index, ok := m.itemAt(listTop)
		if !ok || index != 0 {
			t.Errorf("with listTop %d, the first list row selected (%d, %v), want (0, true)", listTop, index, ok)
		}
		if _, ok := m.itemAt(listTop - 1); ok {
			t.Errorf("with listTop %d, the row above the list selected a tunnel", listTop)
		}
	}
}

func TestItemAtWithNoTunnels(t *testing.T) {
	m := modelWithItems(0, 4)

	if _, ok := m.itemAt(4); ok {
		t.Error("an empty list reported a tunnel")
	}
}
