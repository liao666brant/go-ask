package app

import (
	"strings"
	"time"

	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

const (
	panelWidth               = 760
	panelMaxHeight           = 558
	translationResultsHeight = 276
)

func label(s, class string) core.Node { return core.Text{Content: s, Class: class} }
func box(class string, children ...core.Node) core.View {
	return core.View{Class: class, Children: children}
}
func glyph(svg *core.SVG, class string) core.Node {
	return core.Icon{SVG: svg, Class: "size-4 " + class}
}
func action(name string, svg *core.SVG, primary bool, fn func()) base.Button {
	bg, fg := "border bg-surface "+divider+" "+buttonStates, ink
	if primary {
		bg = "bg-accent enabled:hover:bg-accent-hover enabled:active:bg-accent-pressed " + focusRing
		fg = "text-accent-text"
	}
	var content []core.Node
	if svg != nil {
		content = append(content, glyph(svg, fg))
	}
	content = append(content, label(name, "text-sm font-medium "+fg))
	return base.Button{Label: name, Class: "flex-row gap-2 px-3 h-10 rounded-md items-center justify-center disabled:opacity-50 " + bg, OnPress: func(core.PressEvent) { fn() }, Children: content}
}
func iconButton(name string, svg *core.SVG, fn func()) core.Node {
	return base.Button{Label: name, Tooltip: label(name, "text-sm"), Class: "size-9 items-center justify-center rounded-md " + buttonStates, OnPress: func(core.PressEvent) { fn() }, Children: []core.Node{glyph(svg, "")}}
}

func copyFeedback(cx *core.Scope) (func(string), core.Node) {
	copied := core.NewSignal(cx, false)
	var stop func() bool
	copyText := func(value string) {
		cx.SetClipboard(value)
		if stop != nil {
			stop()
		}
		copied.Set(true)
		stop = cx.AfterFunc(2*time.Second, func() { copied.Set(false) })
	}
	return copyText, core.Text{Role: core.RoleStatus, Class: "text-xs font-semibold " + accent, ContentFunc: func() string {
		if copied.Get() {
			return "已复制"
		}
		return ""
	}}
}

func modeButton(name string, icon *core.SVG, selected func() bool, activate func()) core.Node {
	return base.Button{
		Label: name, Selected: selected, OnPress: func(core.PressEvent) { activate() },
		Class:    "flex-row gap-2 h-9 px-4 rounded-md items-center justify-center text-text-muted " + buttonStates + " " + selectedFill + " aria-selected:text-accent",
		Children: []core.Node{glyph(icon, ""), label(name, "text-sm font-semibold")},
	}
}

func brandMark() core.Node {
	return box("size-8 rounded-md items-center justify-center bg-accent", glyph(lucide.Languages(), "text-accent-text"))
}

func (a *application) root(cx *core.Scope) core.Node {
	cx.SetTheme(workbenchTheme)
	a.host.content = cx
	if a.cx == nil {
		a.initUI(cx)
	}
	pinned := a.pinned
	canPin := func() bool { return desktop.CanSetWindowLevel(cx) }
	cx.Effect(func() {
		if !canPin() {
			return
		}
		level := desktop.WindowLevelNormal
		if pinned.Get() {
			level = desktop.WindowLevelFloating
		}
		desktop.SetWindowLevel(cx, level)
	})
	const pinUnavailable = "当前桌面不支持置顶"
	pinLabel := func() string {
		if pinned.Get() {
			return "取消置顶"
		}
		return "置顶"
	}

	cx.OnKeyWith("ctrl+tab", core.KeyOptions{When: core.KeyAlways}, func() {
		if a.mode.Get() == "translate" {
			a.switchMode("chat")
		} else {
			a.switchMode("translate")
		}
	})
	cx.OnKeyWith("ctrl+n", core.KeyOptions{When: core.KeyAlways}, func() {
		if a.page.Get() == "work" && a.mode.Get() == "chat" {
			a.newConversation()
		}
	})
	// The panel can hide before its backend sees key-up. Accept the next Escape
	// even if that backend still classifies it as a repeat.
	cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways, Repeat: true}, func() { a.host.hide() })
	window := box("bg-background", core.Dynamic(func(cx *core.Scope) core.Node {
		page, mode := a.page.Get(), a.mode.Get()
		header := box("flex-row items-center gap-1 px-4 py-2 bg-surface border-b "+divider,
			brandMark(), label("Go Ask", "ml-2 mr-5 font-semibold text-lg "+ink))
		if page == "work" {
			header.Children = append(header.Children,
				box("flex-row items-center gap-1 p-1 "+inset,
					modeButton("翻译", lucide.Languages(), func() bool { return a.mode.Get() == "translate" }, func() { a.switchMode("translate") }),
					modeButton("问答", lucide.MessageCircle(), func() bool { return a.mode.Get() == "chat" }, func() { a.switchMode("chat") }),
				),
			)
		} else {
			header.Children = append(header.Children, action("返回", lucide.ChevronLeft(), false, func() { a.switchMode(a.mode.Get()) }))
		}
		header.Children = append(header.Children, core.View{Class: "flex-1 h-10", Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOn}})
		if page == "work" {
			header.Children = append(header.Children,
				iconButton("历史会话", lucide.Clock3(), func() {
					a.note.Set("")
					a.page.Set("history")
				}),
				iconButton("设置", lucide.Settings2(), a.host.openSettings),
				box("mx-1 h-5 w-px bg-border"),
			)
		}
		header.Children = append(header.Children,
			core.Show{When: func() bool { return !canPin() }, Then: func(*core.Scope) core.Node {
				return label(pinUnavailable, "text-xs "+muted)
			}},
			base.Button{
				LabelFunc: pinLabel,
				Tooltip:   core.Text{Class: "text-sm", ContentFunc: pinLabel},
				Disabled:  func() bool { return !canPin() },
				Selected:  pinned.Get,
				Class:     "size-9 items-center justify-center rounded-md disabled:opacity-50 " + buttonStates + " " + selectedFill + " aria-selected:text-accent",
				OnPress:   func(core.PressEvent) { a.host.togglePanelPin() },
				Children:  []core.Node{glyph(lucide.Pin(), "")},
			},
			iconButton("收起窗口", lucide.Minus(), a.host.hide),
		)
		var body core.Node
		switch {
		case page == "history":
			body = a.historyView(cx)
		case mode == "chat":
			body = a.chatWorkspace(cx)
		default:
			body = a.translationWorkspace(cx)
		}
		panel := box("shrink", header, body)
		if page == "work" && mode == "chat" {
			// Keep the composer in one place as a streamed reply grows.
			panel.Style.Height = core.Pt(panelMaxHeight)
		}
		return panel
	}))
	window.Style.Width = core.Pt(panelWidth)
	return window
}
func (a *application) input(name string, draft *core.Signal[string], placeholder string, height core.Points, submit func()) ui.Input {
	return ui.Input{Ref: &a.editor, Label: name, AutoFocus: true, Style: core.Style{Height: core.Pt(height), FontSize: 16}, Value: draft.Get, OnChange: draft.Set, Placeholder: placeholder,
		Multiline: true, SubmitBehavior: core.SubmitBehaviorSubmit, OnSubmit: func(text string) { draft.Set(text); submit() }}
}

func (a *application) panelNotice() core.Node {
	return core.Show{When: func() bool { return a.note.Get() != "" }, Then: func(*core.Scope) core.Node {
		return box("px-3 py-2 rounded-md bg-inset",
			core.Text{Role: core.RoleStatus, ContentFunc: a.note.Get, Class: "text-sm " + warning})
	}}
}

func shortcutBar(verb string, feedback core.Node) core.Node {
	footer := box("flex-row flex-wrap gap-x-4 gap-y-1 items-center px-1 min-h-5")
	for _, shortcut := range [][2]string{{"Enter", verb}, {"Shift+Enter", "换行"}, {"Ctrl+Tab", "切换"}, {"Esc", "收起"}} {
		footer.Children = append(footer.Children, box("flex-row items-center gap-1",
			box("h-5 px-1 rounded border bg-surface justify-center "+divider, label(shortcut[0], "font-mono text-xs "+muted)),
			label(shortcut[1], "text-xs "+muted),
		))
	}
	footer.Children = append(footer.Children, core.View{Class: "flex-1"}, feedback)
	return footer
}

func (a *application) translationWorkspace(cx *core.Scope) core.Node {
	copyText, feedback := copyFeedback(cx)
	translate := action("翻译", lucide.ArrowDown(), true, a.translate)
	translate.TestID = "translation-submit"
	translate.Disabled = func() bool { return strings.TrimSpace(a.query.Get()) == "" }
	language := func(name, first, title string, value *core.Signal[string]) core.Node {
		return ui.Select[string]{Label: name, Options: []ui.Option[string]{
			{Value: first, Label: title}, {Value: "zh", Label: "中文"}, {Value: "en", Label: "英语"},
			{Value: "ja", Label: "日语"}, {Value: "fr", Label: "法语"}, {Value: "de", Label: "德语"},
		}, Value: value.Get, OnChange: value.Set, Style: core.Style{Width: core.Pt(128), Height: core.Pt(36)}}
	}
	// Keep the editor's hit area above scrolled result rows.
	composer := box("z-10 p-3 gap-2 "+sheet,
		a.input("翻译原文", a.query, "输入单词或粘贴一段文字…", 84, a.translate),
		box("flex-row items-center gap-2",
			label("源语言", "text-xs "+muted),
			language("源语言", "auto", "自动识别", a.source),
			label("目标语言", "ml-2 text-xs "+muted),
			language("目标语言", "smart", "中英互译", a.target),
			core.Text{Class: "flex-1 ml-1 text-xs " + muted, ContentFunc: func() string {
				if strings.TrimSpace(a.query.Get()) == "" || a.target.Get() != "smart" {
					return ""
				}
				if targetLanguage(a.source.Get(), "smart", a.query.Get()) == "en" {
					return "译为英语"
				}
				return "译为中文"
			}},
			translate,
		),
	)
	var resultsScroll core.Ref
	queryID := a.translationID
	cx.Effect(func() {
		a.resultsVersion.Get()
		if queryID != a.translationID {
			queryID = a.translationID
			cx.Post(func() { resultsScroll.ScrollTo(core.Point{}, false) })
		}
	})
	return box("shrink px-4 pt-4 pb-3 gap-3", composer, a.panelNotice(),
		core.Scroll{
			Label: "翻译结果", Ref: &resultsScroll,
			// Keep the editor and keyboard hints outside the scroll viewport.
			Style: core.Style{MaxHeight: core.Pt(translationResultsHeight)},
			Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
				a.resultsVersion.Get()
				return a.translationResults(copyText)
			})},
		}, shortcutBar("翻译", feedback))
}
func (a *application) translationResults(copyText func(string)) core.Node {
	var rows []core.Node
	if a.found {
		e := a.entry
		query := a.submitted
		headword := box("gap-1", core.Text{Content: e.Word, Class: "text-3xl font-semibold " + ink,
			Style: core.Style{FontFamily: "Georgia, serif"}})
		if s := strings.TrimSpace(e.Phonetic + "  " + e.POS); s != "" {
			headword.Children = append(headword.Children, label(s, "text-sm "+muted))
		}
		rows = append(rows, box("p-4 pr-5 gap-3 "+sheet,
			label("离线英汉词典", "text-xs font-semibold "+accent),
			headword,
			core.Text{Content: strings.TrimSpace(e.Translation), Selectable: core.SelectableOn, Class: "text-base leading-6 " + ink},
			box("flex-row gap-2", action("在线对照", nil, false, func() {
				a.startTranslation(query, true)
			}), iconButton("复制释义", lucide.Copy(), func() { copyText(e.Translation) })),
		))
	}
	for i, r := range a.translations {
		provider, _ := service.ProviderByID(r.provider)
		var actions []core.Node
		value, tone := r.value, ink
		if r.pending {
			value, tone = "翻译中…", muted
		} else if r.failure != "" {
			value, tone = r.failure, warning
			actions = append(actions, action("重试", lucide.RotateCcw(), false, func() {
				if a.translations[i].pending {
					return
				}
				a.translations[i].pending = true
				a.resultsVersion.Update(increment)
				a.requestTranslation(a.translateCtx, i)
			}))
		} else {
			actions = append(actions, action("就此提问", lucide.MessageCircle(), false, func() { a.askAbout(r.value) }), iconButton("复制译文", lucide.Copy(), func() { copyText(r.value) }))
		}
		rows = append(rows, box("px-4 pb-4 pt-2 gap-1 "+sheet,
			box("flex-row items-center justify-between gap-2 min-h-10",
				label(strings.TrimSuffix(provider.Name, "（内置）"), "flex-1 min-w-0 text-sm font-semibold "+ink),
				box("flex-row items-center gap-1", actions...)),
			core.Text{Content: value, Selectable: core.SelectableOn, Class: "text-base leading-6 " + tone},
		))
	}
	if len(rows) == 0 {
		if a.submitted.text != "" {
			return core.View{}
		}
		return box("flex-row items-center gap-3 px-4 py-4 "+sheet,
			box("size-9 rounded-md bg-inset items-center justify-center", glyph(lucide.Languages(), accent)),
			box("flex-1 gap-1", label("输入文字后按 Enter 开始翻译", "text-sm font-semibold "+ink),
				label("单个英文词可优先查离线词典", "text-sm "+muted)))
	}
	if !a.found {
		rows = append([]core.Node{box("flex-row items-center gap-2 px-1",
			label("原文", "text-xs font-semibold "+accent),
			core.Text{Content: a.submitted.text, MaxLines: 1, Class: "flex-1 min-w-0 text-sm " + muted},
		)}, rows...)
	}
	return box("gap-2", rows...)
}
