package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Moving through the list (Batch 8, A3). The list's own keys already move by
// row and by page; Quarter changes what happens at the edges:
//   - ↑ ↓ stay on the page: past its last game, back to its first, and the
//     other way round. Only ← → change pages.
//   - ← → wrap: past the last page, back to the first, and the other way
//     round, keeping the same row on the page where it has one.
//
// wrapMove reports whether it handled msg; everything else is the list's.
func (m *Model) wrapMove(msg tea.KeyPressMsg) bool {
	n := len(m.list.VisibleItems())
	if n == 0 {
		return false
	}
	p := &m.list.Paginator
	start, end := p.GetSliceBounds(n) // this page's games
	lastPage := max(p.TotalPages-1, 0)
	switch {
	case key.Matches(msg, m.list.KeyMap.CursorUp) && m.list.Index() == start:
		m.list.Select(end - 1)
	case key.Matches(msg, m.list.KeyMap.CursorDown) && m.list.Index() == end-1:
		m.list.Select(start)
	case key.Matches(msg, m.list.KeyMap.PrevPage) && p.Page == 0:
		m.list.Select(pageRow(lastPage, m.list.Cursor(), p.PerPage, n))
	case key.Matches(msg, m.list.KeyMap.NextPage) && p.Page == lastPage:
		m.list.Select(pageRow(0, m.list.Cursor(), p.PerPage, n))
	default:
		return false
	}
	return true
}

// pageRow is the index of row on page, or of the page's last game when the
// page is shorter (the last page usually is).
func pageRow(page, row, perPage, n int) int {
	return min(page*perPage+row, n-1)
}
