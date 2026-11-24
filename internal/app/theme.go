package app

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
)

const (
	ink          = "text-text"
	muted        = "text-text-muted"
	faint        = "text-faint"
	accent       = "text-accent"
	warning      = "text-warning"
	danger       = "text-danger"
	divider      = "border-border"
	subtleBorder = "border-line"
	inset        = "rounded-lg bg-inset"
	sheet        = "rounded-xl border bg-surface " + divider
	cardSubtle   = "rounded-xl border bg-surface-subtle " + subtleBorder
	noticeBox    = "rounded-lg border border-warn-border bg-warn-tint"
	focusRing    = "focus-visible:ring-2 focus-visible:ring-focus"
	buttonStates = "enabled:hover:bg-surface-hover enabled:active:bg-surface-pressed " + focusRing
	selectedFill = "aria-selected:bg-accent-tint"
)

// cardShadow gives elevated cards subtle depth without muddy heavy drop shadows.
func cardShadow(cx *core.Scope) core.Style {
	if cx.Dark() {
		return core.Style{
			BorderCurve: core.BorderCurveContinuous,
			Shadows: []core.BoxShadow{{
				Offset: core.Point{Y: 2},
				Blur:   8,
				Color:  core.RGBA(0x000000, 0.28),
			}},
		}
	}
	return core.Style{
		BorderCurve: core.BorderCurveContinuous,
		Shadows: []core.BoxShadow{{
			Offset: core.Point{Y: 1},
			Blur:   3,
			Color:  core.RGBA(0x161918, 0.05),
		}, {
			Offset: core.Point{Y: 4},
			Blur:   12,
			Color:  core.RGBA(0x161918, 0.04),
		}},
	}
}

var workbenchTheme = func() *core.Theme {
	theme := core.DefaultTheme()
	theme.Radius = 10
	for _, spec := range []struct {
		palette *core.Palette
		dark    bool
		bg      uint32
		surface uint32
		subtle  uint32
		inset   uint32
		text    uint32
		muted   uint32
		faint   uint32
		border  uint32
		line    uint32
		accent  uint32
		onAcc   uint32
		tint    uint32
		hover   uint32
		pressed uint32
		warn    uint32
		danger  uint32
		success uint32
	}{
		{
			// Light: Editorial Warm Alabaster + Deep Pine-Teal (harmonized with app icon, 92% neutral)
			palette: &theme.Light,
			dark:    false,
			bg:      0xf6f6f4,
			surface: 0xffffff,
			subtle:  0xfafaf8,
			inset:   0xefefe9,
			text:    0x161918,
			muted:   0x5b6360,
			faint:   0x858d8a,
			border:  0xe4e5e0,
			line:    0xededeb,
			accent:  0x0f6b5e,
			onAcc:   0xffffff,
			tint:    0xe3f2ef,
			hover:   0xf0f1ed,
			pressed: 0xe6e8e3,
			warn:    0xb45309,
			danger:  0xc2410c,
			success: 0x0e7a5f,
		},
		{
			// Dark: Obsidian Carbon + Calm Pine-Jade (high contrast, zero swamp cast)
			palette: &theme.Dark,
			dark:    true,
			bg:      0x111312,
			surface: 0x181b1a,
			subtle:  0x1d211f,
			inset:   0x212624,
			text:    0xf1f3f2,
			muted:   0x9ca5a1,
			faint:   0x6d7773,
			border:  0x2b312f,
			line:    0x242927,
			accent:  0x38c6b4,
			onAcc:   0x07201d,
			tint:    0x16332e,
			hover:   0x232826,
			pressed: 0x2b312f,
			warn:    0xf5a65b,
			danger:  0xf0786c,
			success: 0x45c99f,
		},
	} {
		p := spec.palette
		p.Background = core.RGB(spec.bg)
		p.Surface = core.RGB(spec.surface)
		p.SurfaceHover = core.RGB(spec.hover)
		p.SurfacePressed = core.RGB(spec.pressed)
		p.Border = core.RGB(spec.border)
		p.Text = core.RGB(spec.text)
		p.TextMuted = core.RGB(spec.muted)
		p.Accent = core.RGB(spec.accent)
		if spec.dark {
			p.AccentHover = ui.Mix(p.Accent, p.Text, 0.14)
			p.AccentPressed = ui.Mix(p.Accent, p.Background, 0.14)
		} else {
			p.AccentHover = ui.Mix(p.Accent, p.Text, 0.14)
			p.AccentPressed = ui.Mix(p.Accent, p.Text, 0.24)
		}
		p.AccentText = core.RGB(spec.onAcc)
		p.Warning = core.RGB(spec.warn)
		p.Danger = core.RGB(spec.danger)
		p.Success = core.RGB(spec.success)
		p.Selection = core.RGB(spec.tint)
		p.Focus = p.Accent.Faded(0.65)
		p.Scrollbar = p.TextMuted.Faded(0.35)

		warnTint := ui.Mix(p.Surface, p.Warning, 0.09)
		warnBorder := ui.Mix(p.Surface, p.Warning, 0.28)
		accentBorder := ui.Mix(p.Surface, p.Accent, 0.24)
		codeBg := core.RGB(spec.inset)
		codeHeader := ui.Mix(codeBg, p.Border, 0.35)
		if spec.dark {
			warnTint = ui.Mix(p.Surface, p.Warning, 0.13)
			warnBorder = ui.Mix(p.Surface, p.Warning, 0.32)
			accentBorder = ui.Mix(p.Surface, p.Accent, 0.30)
			codeBg = core.RGB(0x141716)
			codeHeader = core.RGB(0x1c201e)
		}

		for name, color := range map[string]core.Color{
			"inset":          core.RGB(spec.inset),
			"surface-subtle": core.RGB(spec.subtle),
			"field":          core.RGB(spec.subtle),
			"raised":         p.Surface,
			"fieldBorder":    p.Border,
			"line":           core.RGB(spec.line),
			"faint":          core.RGB(spec.faint),
			"code":           p.Text,
			"code-bg":        codeBg,
			"code-header":    codeHeader,
			"accent-tint":    p.Selection,
			"accent-border":  accentBorder,
			"accentFill":     p.Accent,
			"warn-tint":      warnTint,
			"warn-border":    warnBorder,
			"info":           p.Accent,
			"down":           p.Danger,
			"light":          p.Accent,
			"onLight":        p.AccentText,
			"track":          core.RGB(spec.faint),
			"knob":           p.AccentText,
		} {
			p.SetColor(name, color)
		}
	}
	return theme
}()
