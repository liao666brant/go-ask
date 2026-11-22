package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"goask/internal/selection"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

type desktopHost struct {
	app              *application
	scope            *core.Scope
	content          *core.Scope
	panel            *desktop.Window
	panelHidesOnBlur bool
	stopIdle         func() bool
	hotkey           string
	activeKey        *core.Scope
	changing         bool
	showTicket       uint64
}

func runDesktop(a *application, trayIcon []byte) {
	nsgo.Run(func(cx *core.Scope) {
		h := &desktopHost{app: a, scope: cx}
		a.host = h
		desktop.SetShutdownMode(cx, desktop.ShutdownOnExplicit)
		a.initUI(cx)
		desktop.NewTray(cx, desktop.TrayOptions{
			Icon: trayIcon, Tooltip: "Go Ask", OnClick: func() { h.show("") },
			Menu: func() []desktop.MenuItem {
				return []desktop.MenuItem{
					{Label: "打开 Go Ask", OnSelect: func() { h.show("") }},
					{Label: "设置", OnSelect: h.openSettings},
					{Label: "关于 Go Ask", OnSelect: h.openAbout},
					{Separator: true},
					{Label: "退出", OnSelect: h.quit},
				}
			},
		})
		desktop.OnReopen(cx, func() { h.show("") })
		h.changeHotkey(a.desktopHotkey(), func(err error) {
			if err != nil {
				a.onHotkeyError(err)
			}
		})
	}, nsgo.InstanceKey("go-ask"))
}

func (h *desktopHost) openSettings() {
	h.showTicket++
	h.scope.Run(func() {
		desktop.OpenWindow(h.scope, desktop.WindowOptions{
			Key: "settings", Title: "Go Ask 设置", Width: 648, Height: 600,
			Constraints: desktop.WindowConstraints{MinWidth: 400, MinHeight: 340},
		}, h.app.settingsView)
	})
}

func (h *desktopHost) openAbout() {
	h.showTicket++
	h.scope.Run(func() {
		desktop.OpenWindow(h.scope, desktop.WindowOptions{
			Key: "about", Title: "关于 Go Ask", Width: 400, Height: 248,
			Constraints: desktop.WindowConstraints{NoResize: true},
		}, aboutView)
	})
}

func (h *desktopHost) hide() {
	h.showTicket++
	h.panel.Hide()
}

func (h *desktopHost) closePanel() {
	h.panel.Close()
	h.panel, h.content = nil, nil
	h.app.editor = core.Ref{}
}

func (h *desktopHost) cancelPanelRelease() {
	if h.stopIdle != nil {
		h.stopIdle()
		h.stopIdle = nil
	}
}

func (h *desktopHost) schedulePanelRelease() {
	h.cancelPanelRelease()
	seconds := h.app.pref.PanelIdleSeconds
	if seconds <= 0 || h.panel == nil || h.panel.Visible() {
		return
	}
	h.scope.Run(func() {
		h.stopIdle = h.scope.AfterFunc(time.Duration(seconds)*time.Second, func() {
			h.stopIdle = nil
			h.closePanel()
		})
	})
}

func (h *desktopHost) quit() {
	if h.app.cx != nil {
		h.app.stopChat()
	}
	h.scope.Quit()
}

func (h *desktopHost) show(text string) {
	h.showTicket++
	h.cancelPanelRelease()
	h.ensurePanel()
	// NSGo resets its blur tracking when asked to focus an already focused panel.
	if !h.panelHidesOnBlur || !h.panel.Visible() || desktop.FrontmostApp(h.scope).PID != os.Getpid() {
		h.panel.Focus()
	}
	h.app.onDesktopShow(text)
}

func (h *desktopHost) ensurePanel() {
	hideOnBlur := h.panelBlurEnabled()
	if h.panel != nil && h.panelHidesOnBlur != hideOnBlur {
		h.closePanel()
	}
	if h.panel == nil {
		h.openPanel(hideOnBlur)
	}
}

func (h *desktopHost) panelBlurEnabled() bool {
	return h.app.pref.HidePanelOnBlur && !h.app.pinned.GetUntracked()
}

func (h *desktopHost) openPanel(hideOnBlur bool) {
	h.scope.Run(func() {
		panel := desktop.OpenPanel(h.scope, desktop.PanelOptions{
			Width: panelWidth, Height: panelMaxHeight, FitContent: true, HideOnBlur: hideOnBlur,
		}, h.app.root)
		h.panel, h.panelHidesOnBlur = panel, hideOnBlur
		h.content.Run(func() {
			wasVisible := false
			h.content.Effect(func() {
				visible := panel.Visible()
				if h.panel != panel {
					return
				}
				if visible {
					h.cancelPanelRelease()
				} else if wasVisible {
					h.schedulePanelRelease()
				}
				wasVisible = visible
			})
		})
		panel.OnClosing(h.scope, func(close *desktop.WindowClosing) {
			close.Cancel()
			h.hide()
		})
	})
}

func (h *desktopHost) togglePanelPin() {
	h.app.pinned.Set(!h.app.pinned.GetUntracked())
	hideOnBlur := h.panelBlurEnabled()
	if !h.panel.Visible() || h.panelHidesOnBlur == hideOnBlur {
		return
	}
	oldPanel := h.panel
	var frame desktop.WindowFrame
	if desktop.CanPlaceWindows(h.content) {
		frame = h.content.Env().WindowFrame()
	}
	h.openPanel(hideOnBlur)
	h.panel.Focus()
	if frame.Move {
		desktop.SetWindowFrame(h.content, frame)
	}
	// Let the replacement take focus before destroying the foreground window.
	h.scope.Post(oldPanel.Close)
}

// Capture the selection before giving our panel focus. UI Automation providers
// run in other processes, so a slow provider must not block the launcher.
func (h *desktopHost) fromHotkey() {
	if desktop.FrontmostApp(h.scope).PID == os.Getpid() {
		h.show("")
		return
	}
	h.showTicket++
	ticket := h.showTicket
	ctx, cancel := context.WithTimeout(h.scope.Context(), 700*time.Millisecond)
	result := make(chan string, 1)
	go func() { result <- selection.Text(ctx) }()
	go func() {
		defer cancel()
		var text string
		select {
		case text = <-result:
		case <-ctx.Done():
		}
		h.scope.Post(func() {
			if h.showTicket == ticket {
				h.show(text)
			}
		})
	}()
}

func (h *desktopHost) changeHotkey(requested string, done func(error)) {
	key, err := normalizeHotkey(requested)
	if err != nil {
		done(err)
		return
	}
	if h.changing {
		done(errors.New("正在等待系统确认快捷键"))
		return
	}
	if key == h.hotkey {
		done(nil)
		return
	}
	h.changing = true
	// Register under the app scope, so replacing the settings view never removes
	// its global registration. Each binding has a disposable owner of its own.
	h.scope.Run(func() {
		candidate := h.scope.Child()
		candidate.Run(func() {
			binding := desktop.RegisterHotKey(candidate, key, h.fromHotkey)
			var stop func()
			stop = candidate.Effect(func() {
				state := binding.State()
				if state == desktop.HotKeyPending {
					return
				}
				h.scope.Post(func() {
					if candidate.Disposed() || !h.changing {
						return
					}
					stop()
					h.changing = false
					switch state {
					case desktop.HotKeyRegistered:
						old := h.activeKey
						h.activeKey, h.hotkey = candidate, key
						if old != nil {
							old.Dispose()
						}
						done(nil)
					case desktop.HotKeyTaken:
						candidate.Dispose()
						done(fmt.Errorf("系统未分配快捷键 %s，可能已被占用或未获授权", key))
					default:
						candidate.Dispose()
						done(fmt.Errorf("当前桌面不支持全局快捷键 %s，可从应用菜单再次打开 Go Ask", key))
					}
				})
			})
		})
	})
}

// NSGo ignores unknown modifiers while normalizing. Reject a typo such as
// "crtl+q" here so that it cannot accidentally take the unmodified Q key.
func normalizeHotkey(input string) (string, error) {
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input), " ", "")), "+")
	key := parts[len(parts)-1]
	if key == "" {
		return "", errors.New("请输入完整的唤起快捷键")
	}
	modifiers := make(map[string]bool)
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "ctrl", "control", "mod":
			modifiers["ctrl"] = true
		case "alt", "option":
			modifiers["alt"] = true
		case "shift":
			modifiers["shift"] = true
		case "meta", "super", "win", "cmd", "command":
			modifiers["meta"] = true
		default:
			return "", fmt.Errorf("无法识别快捷键修饰键 %q", part)
		}
	}
	var normalized []string
	for _, modifier := range []string{"ctrl", "alt", "shift", "meta"} {
		if modifiers[modifier] {
			normalized = append(normalized, modifier)
		}
	}
	return strings.Join(append(normalized, key), "+"), nil
}
