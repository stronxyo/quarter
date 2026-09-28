package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/stronxyo/quarter/internal/config"
)

// On a Sixel terminal the band stays blank and the logo is sent as a
// picture at the band's top-left cell.
func TestLogoAsPicture(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	m, _ := withLogos(t, logos, config.Settings{})
	m, cmd := m.Update(uv.PrimaryDeviceAttributesEvent{62, 4, 22})
	m = pump(m, cmd)
	m, cmd = m.Update(uv.CellSizeEvent{Width: 10, Height: 20})
	m = pump(m, cmd)
	if hasGameLogo(paneText(m)) {
		t.Error("the band should be blank: the picture is drawn over it")
	}
	sig := m.(Model).pictureWant
	if !sig.ready {
		t.Fatalf("the picture should be ready: %+v", sig)
	}
	_, cmd = m.Update(pictureMsg{sig})
	raw, ok := cmd().(tea.RawMsg)
	if !ok {
		t.Fatal("drawing the picture should send it raw")
	}
	s := fmt.Sprint(raw.Msg)
	at := fmt.Sprintf("\x1b[%d;%dH\x1bP", logoPadTop+1, sig.x+1)
	if !strings.Contains(s, at) || !strings.HasSuffix(s, "\x1b\\\x1b8") {
		t.Errorf("the picture should start at the band and end the Sixel: %q...", s[:min(len(s), 80)])
	}
	if want := fmt.Sprintf("\"1;1;%d;%d", sig.cols*10, sig.rows*20); !strings.Contains(s, want) {
		t.Errorf("the picture should fill the band in pixels, want %s", want)
	}
}
