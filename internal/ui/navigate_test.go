package ui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// paged is an 80x24 model with n games, so several pages.
func paged(t *testing.T, n int) Model {
	t.Helper()
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("Game %03d", i)
	}
	return ready(t, names...)
}

func keyPress(m Model, code rune) Model {
	next, _ := m.Update(tea.KeyPressMsg{Code: code})
	return next.(Model)
}

// where is the selection: page, and index in the whole list.
func where(m Model) (page, index int) {
	return m.list.Paginator.Page, m.list.Index()
}

// ↑ ↓ wrap within the page; only ← → change it.
func TestMoveStaysOnPage(t *testing.T) {
	m := paged(t, 100)
	per := m.list.Paginator.PerPage
	if m.list.Paginator.TotalPages < 3 {
		t.Fatalf("the test needs several pages, got %d", m.list.Paginator.TotalPages)
	}

	m = keyPress(m, tea.KeyUp)
	if p, i := where(m); p != 0 || i != per-1 {
		t.Errorf("↑ on the first game: page %d game %d, want page 0 game %d", p, i, per-1)
	}
	m = keyPress(m, tea.KeyDown)
	if p, i := where(m); p != 0 || i != 0 {
		t.Errorf("↓ on the page's last game: page %d game %d, want page 0 game 0", p, i)
	}
	m = keyPress(m, tea.KeyDown)
	if _, i := where(m); i != 1 {
		t.Errorf("↓ inside the page should move down one, got game %d", i)
	}

	// The same on a middle page.
	m = keyPress(m, tea.KeyRight)
	m = keyPress(m, tea.KeyUp) // row 1 to row 0
	m = keyPress(m, tea.KeyUp)
	if p, i := where(m); p != 1 || i != 2*per-1 {
		t.Errorf("↑ on page 1's first game: page %d game %d, want page 1 game %d", p, i, 2*per-1)
	}
}

// ← → wrap from the first page to the last and back, keeping the row where
// the last, shorter page has it.
func TestPagesWrap(t *testing.T) {
	m := paged(t, 100)
	per, pages := m.list.Paginator.PerPage, m.list.Paginator.TotalPages
	last := (pages - 1) * per
	short := 100 - last // games on the last page
	if short >= per {
		t.Fatalf("the test needs a shorter last page, got %d of %d", short, per)
	}

	m = keyPress(m, tea.KeyLeft)
	if p, i := where(m); p != pages-1 || i != last {
		t.Errorf("← on the first page: page %d game %d, want page %d game %d", p, i, pages-1, last)
	}
	m = keyPress(m, tea.KeyRight)
	if p, i := where(m); p != 0 || i != 0 {
		t.Errorf("→ on the last page: page %d game %d, want page 0 game 0", p, i)
	}

	// A row the last page doesn't have lands on its last game.
	for range per - 1 {
		m = keyPress(m, tea.KeyDown)
	}
	m = keyPress(m, tea.KeyLeft)
	if p, i := where(m); p != pages-1 || i != 99 {
		t.Errorf("← from row %d: page %d game %d, want the last game, 99", per-1, p, i)
	}

	// The last page's own edges wrap within it.
	m = keyPress(m, tea.KeyDown)
	if _, i := where(m); i != last {
		t.Errorf("↓ on the last game: game %d, want the last page's first, %d", i, last)
	}
}

// With one page, ← → stay and ↑ ↓ wrap around it.
func TestOnePage(t *testing.T) {
	m := paged(t, 3)
	for _, k := range []rune{tea.KeyLeft, tea.KeyRight} {
		if p, i := where(keyPress(m, k)); p != 0 || i != 0 {
			t.Errorf("%q on the only page moved to page %d game %d", k, p, i)
		}
	}
	if _, i := where(keyPress(m, tea.KeyUp)); i != 2 {
		t.Errorf("↑ on the first of three games: game %d, want 2", i)
	}
}
