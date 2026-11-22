package app

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
)

const (
	ink          = "text-text"
	muted        = "text-text-muted"
	accent       = "text-accent"
	warning      = "text-warning"
	divider      = "border-border"
	inset        = "rounded-md bg-inset"
	sheet        = "rounded-lg border bg-surface " + divider
	focusRing    = "focus-visible:ring-2 focus-visible:ring-focus"
	buttonStates = "enabled:hover:bg-surface-hover enabled:active:bg-surface-pressed " + focusRing
	selectedFill = "aria-selected:bg-accent-tint"
)

var workbenchTheme = func() *core.Theme {
	theme := core.DefaultTheme()
	theme.Radius = 8
	for _, colors := range []struct {
		palette                                                                                       *core.Palette
		background, surface, inset, text, muted, border, accent, onAccent, tint, hover, pressed, warn uint32
	}{
		{&theme.Light, 0xf2f6f5, 0xffffff, 0xeef4f2, 0x152a28, 0x4c6561, 0xd2e0dc, 0x08665b, 0xffffff, 0xd5eee8, 0xe7f2ef, 0xdcece7, 0x963e16},
		{&theme.Dark, 0x131c1b, 0x1b2624, 0x233230, 0xeaf5f2, 0xa9c2bc, 0x3b514d, 0x83ddd0, 0x102d29, 0x24453f, 0x2a3e3a, 0x34504a, 0xffb58e},
	} {
		p := colors.palette
		p.Background = core.RGB(colors.background)
		p.Surface = core.RGB(colors.surface)
		p.SurfaceHover = core.RGB(colors.hover)
		p.SurfacePressed = core.RGB(colors.pressed)
		p.Border = core.RGB(colors.border)
		p.Text = core.RGB(colors.text)
		p.TextMuted = core.RGB(colors.muted)
		p.Accent = core.RGB(colors.accent)
		p.AccentHover = ui.Mix(p.Accent, p.Text, 0.12)
		p.AccentPressed = ui.Mix(p.Accent, p.Text, 0.22)
		p.AccentText = core.RGB(colors.onAccent)
		p.Warning = core.RGB(colors.warn)
		p.Selection = core.RGB(colors.tint)
		p.Focus = p.Accent
		p.Scrollbar = p.TextMuted.Faded(0.42)
		for name, color := range map[string]core.Color{
			"inset": core.RGB(colors.inset), "field": core.RGB(colors.inset), "raised": p.Surface,
			"fieldBorder": p.Border, "line": p.Border, "faint": p.TextMuted, "code": p.Text,
			"accent-tint": p.Selection, "accentFill": p.Accent, "light": p.Accent, "onLight": p.AccentText,
			"track": p.TextMuted, "knob": p.AccentText,
		} {
			p.SetColor(name, color)
		}
	}
	return theme
}()
