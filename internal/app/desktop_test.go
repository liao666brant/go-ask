package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goask/internal/data"
	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestContentPanelPin(t *testing.T) {
	start := func(t *testing.T, canPin bool) *nsgotest.App {
		t.Helper()
		db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		a := &application{db: db}
		a.host = &desktopHost{app: a}
		return nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight), nsgotest.Environment(core.Environment{
			OS: core.OSWindows, Idiom: core.IdiomDesktop, CanSetWindowLevel: canPin,
		}))
	}

	t.Run("supported", func(t *testing.T) {
		panel := start(t, true)
		pin := panel.Find("label:置顶").Bounds()
		hide := panel.Find("label:收起窗口").Bounds()
		if pin.Origin.X+pin.Size.Width >= hide.Origin.X || pin.Origin.Y != hide.Origin.Y {
			t.Fatalf("pin button is not beside hide: %v, %v", pin, hide)
		}
		panel.Focus("label:置顶")
		panel.Key("space")
		panel.Find("label:取消置顶")
		panel.Key("ctrl+tab")
		panel.Find("text:有什么想问的？")
		panel.Find("label:取消置顶")
		panel.Press("label:历史会话")
		panel.Find("label:返回")
		panel.Find("label:取消置顶")
		panel.Press("label:取消置顶")
		panel.Find("label:置顶")
		panel.Press("label:返回")
		panel.Find("label:置顶")
	})

	t.Run("unsupported", func(t *testing.T) {
		panel := start(t, false)
		if panel.Enabled("label:置顶") {
			t.Fatal("pin button enabled without window level support")
		}
		panel.Find("text:当前桌面不支持置顶")
		panel.Find("label:收起窗口")
	})
}

func TestTranslationPanelFitsShortAndLongResults(t *testing.T) {
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{db: db}
	app := nsgotest.StartApp(t, func(cx *core.Scope) {
		a.host = &desktopHost{app: a, scope: cx}
		a.initUI(cx)
	}, nsgotest.Environment(core.Environment{OS: core.OSWindows, Idiom: core.IdiomDesktop}))
	run := func(fn func()) {
		app.Post(fn)
		app.Settle()
	}
	height := func() core.Points {
		var h core.Points
		run(func() { h = a.host.content.Env().Window().Height })
		return h
	}

	run(func() { a.host.show("") })
	emptyHeight := height()
	run(func() {
		a.translations = []translation{
			{provider: service.DeepL, value: "第一家译文"},
			{provider: service.MTran, value: "第二家译文"},
		}
		a.resultsVersion.Update(increment)
	})
	shortHeight := height()
	if emptyHeight <= 0 || shortHeight <= emptyHeight || shortHeight >= panelMaxHeight {
		t.Fatalf("empty and short panels should fit their content: %v, %v", emptyHeight, shortHeight)
	}
	run(func() {
		a.translations[0].value = strings.Repeat("这是一段用来验证滚动的长译文。\n", 60)
		a.resultsVersion.Update(increment)
	})
	longHeight := height()
	if longHeight <= shortHeight || longHeight > panelMaxHeight {
		t.Fatalf("long results should grow within the panel limit, got %v", longHeight)
	}
	layout := &application{db: db}
	layout.host = &desktopHost{app: layout}
	panel := nsgotest.Start(t, layout.root, nsgotest.Size(panelWidth, longHeight))
	emptyEditorHeight := panel.Find("role:textbox").Bounds().Size.Height
	panel.Post(func() {
		layout.translations = a.translations
		layout.resultsVersion.Update(increment)
	})
	panel.Settle()
	scroll := panel.Find("label:翻译结果")
	editor := panel.Find("role:textbox").Bounds()
	source := panel.Find("label:源语言").Bounds()
	target := panel.Find("label:目标语言").Bounds()
	output := scroll.Bounds()
	if editor.Origin.Y+editor.Size.Height > source.Origin.Y || source.Origin.Y != target.Origin.Y ||
		target.Origin.Y+target.Size.Height > output.Origin.Y || output.Size.Width < editor.Size.Width {
		t.Fatalf("full-width results should appear below the editor and language selectors: editor %v, output %v, selectors %v %v", editor, output, source, target)
	}
	if editor.Size.Height != emptyEditorHeight {
		t.Fatal("long translations should leave the height of the source editor unchanged")
	}
	if scroll.Content.Height <= scroll.Bounds().Size.Height {
		t.Fatal("long results must overflow the scroll viewport")
	}
	esc := panel.Find("text:Esc").Bounds()
	if esc.Origin.Y+esc.Size.Height > longHeight {
		t.Fatal("the keyboard hints were clipped by long results")
	}
	panel.ScrollTo("label:翻译结果", core.Point{Y: 200})
	if panel.Find("label:翻译结果").Offset.Y <= 0 {
		t.Fatal("could not scroll the long translation")
	}
	panel.ScrollTo("label:翻译结果", core.Point{Y: scroll.Content.Height})
	panel.Find("text:第二家译文")
	panel.Post(func() {
		previous := layout.translations
		layout.clearTranslation()
		layout.translations = previous
		layout.resultsVersion.Update(increment)
	})
	panel.Settle()
	if panel.Find("label:翻译结果").Offset.Y != 0 {
		t.Fatal("a new query must start at the top of its results")
	}
	run(a.clearTranslation)
	if got := height(); got != emptyHeight {
		t.Fatalf("clearing the results should restore the empty height %v, got %v", emptyHeight, got)
	}
}

func TestHiddenPanelReleasesAndReopens(t *testing.T) {
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{db: db, pref: preferences{PanelIdleSeconds: 10}}
	app := nsgotest.StartApp(t, func(cx *core.Scope) {
		a.host = &desktopHost{app: a, scope: cx}
		a.initUI(cx)
	}, nsgotest.Environment(core.Environment{OS: core.OSWindows}))
	run := func(fn func()) {
		app.Post(fn)
		app.Settle()
	}
	h := a.host
	run(func() {
		h.show("")
		a.pinned.Set(true)
		a.question.Set("草稿")
		h.hide()
	})
	first, firstScope := h.panel, h.content
	if first.Visible() || first.Closed() {
		t.Fatal("hide must leave the first panel mounted")
	}
	app.Advance(9 * time.Second)
	run(func() { h.show("") })
	app.Advance(2 * time.Second)
	if h.panel != first || !first.Visible() {
		t.Fatal("waking before the deadline must reuse the panel")
	}
	run(h.hide)
	app.Advance(10 * time.Second)
	if h.panel != nil || h.content != nil || !first.Closed() || !firstScope.Disposed() {
		t.Fatal("idle timeout did not release the panel and its scope")
	}
	a.cx.Post(func() { a.partial.Set("后台响应") })
	app.Settle()
	run(func() { h.show("hello") })
	if h.panel == first || !h.panel.Visible() || !a.pinned.GetUntracked() ||
		a.question.GetUntracked() != "草稿" || a.partial.GetUntracked() != "后台响应" ||
		a.query.GetUntracked() != "hello" || !a.found {
		t.Fatal("reopened panel lost its app state or incoming selection")
	}
	second := h.panel
	run(func() {
		h.hide()
		a.pref.PanelIdleSeconds = 0
		h.schedulePanelRelease()
	})
	app.Advance(time.Minute)
	if h.panel != second || second.Closed() {
		t.Fatal("disabling release must cancel the pending timer")
	}
}

func TestPanelBlurPolicyKeepsContentAndUsesIdleRelease(t *testing.T) {
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{db: db, pref: preferences{HidePanelOnBlur: true, PanelIdleSeconds: 10}}
	app := nsgotest.StartApp(t, func(cx *core.Scope) {
		a.host = &desktopHost{app: a, scope: cx}
		a.initUI(cx)
	}, nsgotest.Environment(core.Environment{OS: core.OSWindows}))
	run := func(fn func()) {
		app.Post(fn)
		app.Settle()
	}
	h := a.host
	run(func() { h.show("") })
	first := h.panel
	if !h.panelHidesOnBlur {
		t.Fatal("ordinary panel must hide on blur")
	}
	run(func() {
		a.switchMode("chat")
		a.page.Set("history")
		a.question.Set("草稿")
		a.partial.Set("当前回复")
		h.togglePanelPin()
	})
	pinned := h.panel
	if pinned == first || !first.Closed() || h.panelHidesOnBlur || !pinned.Visible() {
		t.Fatal("pinned panel must stay visible on blur")
	}
	if a.page.GetUntracked() != "history" || a.mode.GetUntracked() != "chat" ||
		a.question.GetUntracked() != "草稿" || a.partial.GetUntracked() != "当前回复" {
		t.Fatal("changing panel policy cleared page or chat state")
	}
	run(h.togglePanelPin)
	unpinned := h.panel
	if unpinned == pinned || !h.panelHidesOnBlur {
		t.Fatal("unpinning must restore the native blur policy")
	}
	// A native hide changes Window.Visible without calling the host's hide.
	run(unpinned.Hide)
	app.Advance(9 * time.Second)
	run(func() { h.show("") })
	app.Advance(2 * time.Second)
	if h.panel != unpinned || !unpinned.Visible() {
		t.Fatal("waking a panel hidden by the platform must cancel idle release")
	}
	run(unpinned.Hide)
	app.Advance(10 * time.Second)
	if h.panel != nil || !unpinned.Closed() {
		t.Fatal("hiding through the window did not release the idle panel")
	}
	run(func() {
		a.pref.HidePanelOnBlur = false
		h.show("")
	})
	open := h.panel
	run(h.togglePanelPin)
	if h.panel != open || h.panelHidesOnBlur {
		t.Fatal("turning off the setting must disable native hiding for both pin states")
	}
}

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
