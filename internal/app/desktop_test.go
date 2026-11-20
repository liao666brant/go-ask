package app

import (
	"testing"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestChangingHotkeyKeepsWorkingBindingOnFailure(t *testing.T) {
	h := &desktopHost{}
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		h.scope = cx
		return core.Text{Content: "shortcuts"}
	}, nsgotest.HotKeyTaken("ctrl+h"), nsgotest.HotKeyUnsupported("ctrl+unknown"))
	change := func(key string) error {
		t.Helper()
		var result error
		finished := false
		app.Post(func() {
			h.changeHotkey(key, func(err error) { result, finished = err, true })
		})
		app.Settle()
		if !finished {
			t.Fatalf("no registration result for %s", key)
		}
		return result
	}

	if err := change("Ctrl+Q"); err != nil {
		t.Fatal(err)
	}
	first := h.activeKey
	for _, invalid := range []string{"ctrl+h", "ctrl+unknown", "crtl+q"} {
		if err := change(invalid); err == nil || h.hotkey != "ctrl+q" || first != h.activeKey || first.Disposed() {
			t.Fatalf("binding %s changed the working shortcut: %v", invalid, err)
		}
	}
	if err := change("Shift + Control + Y"); err != nil || h.hotkey != "ctrl+shift+y" || !first.Disposed() {
		t.Fatalf("replacement failed: %v (%s)", err, h.hotkey)
	}
}

// A menu or the lookup toolbar may be disposed as soon as its action returns.
// Windows opened there must remain owned by the application.
func TestUtilityWindowsOutliveTheirLauncher(t *testing.T) {
	a := &application{pref: preferences{Hotkey: "ctrl+q"}}
	app := nsgotest.StartApp(t, func(cx *core.Scope) {
		a.host = &desktopHost{app: a, scope: cx}
	}, nsgotest.Environment(core.Environment{OS: core.OSLinux}))
	app.Post(func() {
		cx := a.host.scope
		launcher := cx.Child()
		launcher.Run(func() { a.host.openSettings(); a.host.openAbout() })
		launcher.Dispose()
		for _, key := range []string{"settings", "about"} {
			built := false
			w := desktop.OpenWindow(cx, desktop.WindowOptions{Key: key}, func(*core.Scope) core.Node {
				built = true
				return core.Text{}
			})
			if built || !w.Visible() {
				t.Errorf("%s did not survive its launcher", key)
			}
			w.Close()
		}
	})
	app.Settle()
}
