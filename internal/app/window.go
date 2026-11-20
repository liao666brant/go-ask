package app

import (
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"goask/internal/service"
)

const (
	ink    = "text-slate-900 dark:text-slate-100"
	muted  = "text-slate-600 dark:text-slate-300"
	accent = "text-indigo-700 dark:text-indigo-300"
	card   = "rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700"
)

func label(s, class string) core.Node { return core.Text{Content: s, Class: class} }
func box(class string, children ...core.Node) core.View {
	return core.View{Class: class, Children: children}
}
func glyph(svg *core.SVG, class string) core.Node {
	return core.Icon{SVG: svg, Class: "size-4 " + class}
}
func action(name string, svg *core.SVG, primary bool, fn func()) core.Node {
	bg, fg := "bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600", ink
	if primary {
		bg = "bg-indigo-600 hover:bg-indigo-700"
		fg = "text-white"
	}
	var content []core.Node
	if svg != nil {
		content = append(content, glyph(svg, fg))
	}
	content = append(content, label(name, "text-sm font-medium "+fg))
	return base.Button{Label: name, Class: "flex-row gap-2 px-3 h-8 rounded-lg items-center justify-center " + bg, OnPress: func(core.PressEvent) { fn() }, Children: content}
}
func iconButton(name string, svg *core.SVG, fn func()) core.Node {
	return base.Button{Label: name, Tooltip: label(name, "text-sm"), Class: "size-8 items-center justify-center rounded-lg hover:bg-slate-200 dark:hover:bg-slate-700", OnPress: func(core.PressEvent) { fn() }, Children: []core.Node{glyph(svg, muted)}}
}
func (a *application) root(cx *core.Scope) core.Node {
	a.initUI(cx)
	lastHeight := core.Points(0)
	cx.Effect(func() {
		a.resultsVersion.Get()
		height := core.Points(510)
		if a.page.Get() == "work" && a.mode.Get() == "translate" && !a.found && len(a.translations) == 0 {
			height = 255
		}
		if height != lastHeight {
			lastHeight = height
			a.host.resize(height)
		}
	})

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
	return box("flex-1 bg-slate-50 dark:bg-slate-900", core.Dynamic(func(cx *core.Scope) core.Node {
		page, mode := a.page.Get(), a.mode.Get()
		header := box("flex-row items-center gap-2 px-4 py-2 border-b border-slate-200 dark:border-slate-700",
			box("size-8 rounded-lg items-center justify-center bg-indigo-600", glyph(lucide.Languages(), "text-white")),
			label("Go Ask", "font-bold text-lg "+ink), core.View{Class: "flex-1 h-8", Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOn}})
		if page == "work" {
			header.Children = append(header.Children, action("翻译", lucide.Languages(), mode == "translate", func() { a.switchMode("translate") }), action("问答", lucide.MessageCircle(), mode == "chat", func() { a.switchMode("chat") }), iconButton("历史会话", lucide.Clock3(), func() { a.page.Set("history") }), iconButton("设置", lucide.Settings2(), a.host.openSettings))
		} else {
			header.Children = append(header.Children, action("返回", lucide.ChevronLeft(), false, func() { a.page.Set("work"); a.focusEditor() }))
		}
		header.Children = append(header.Children, iconButton("收起窗口", lucide.Minus(), a.host.hide))
		var body core.Node
		if page == "history" {
			body = a.historyView(cx)
		} else {
			body = a.workspace(cx, mode)
		}
		panel := box("flex-1", body)
		if !cx.Env().ReduceMotion() {
			fade := core.NewAnimatedValue(cx, 0.35)
			slide := core.Interpolate(fade, core.Interpolation{Input: []float64{0.35, 1}, Output: []float64{4, 0}})
			panel.Style.Animated = &core.AnimatedStyle{Opacity: fade, TranslateY: slide}
			cx.StartAnimation(core.Timing(fade, 1, core.TimingConfig{Duration: 140 * time.Millisecond}), nil)
		}
		return box("flex-1", header, panel)
	}))
}
func (a *application) workspace(cx *core.Scope, mode string) core.Node {
	isChat := mode == "chat"
	draft := a.query
	placeholder := "输入单词或粘贴一段文字…"
	submit := a.translate
	if isChat {
		draft = a.question
		placeholder = "有什么想问的？"
		submit = a.send
	}
	field := core.Input{Ref: &a.editor, AutoFocus: true, Class: "w-full h-16 px-3 py-2 text-base " + ink, Value: draft.Get, OnChange: draft.Set, Placeholder: placeholder,
		Multiline: true, SubmitBehavior: core.SubmitBehaviorSubmit, OnSubmit: func(text string) { draft.Set(text); submit() }}
	var controls []core.Node
	if isChat {
		controls = append(controls, action("新话题", lucide.Plus(), false, a.newConversation))
	} else {
		opts := []ui.Option[string]{{Value: "zh", Label: "中文"}, {Value: "en", Label: "英语"}, {Value: "ja", Label: "日语"}, {Value: "fr", Label: "法语"}, {Value: "de", Label: "德语"}}
		for _, side := range []struct {
			name, first, title string
			s                  *core.Signal[string]
		}{{"源语言", "auto", "自动识别", a.source}, {"目标语言", "smart", "中英互译", a.target}} {
			controls = append(controls, ui.Select[string]{Label: side.name, Options: append([]ui.Option[string]{{Value: side.first, Label: side.title}}, opts...), Value: side.s.Get, OnChange: side.s.Set, Style: core.Style{Width: core.Pt(116), Height: core.Pt(32)}})
			if side.first == "auto" {
				controls = append(controls, glyph(lucide.ArrowUpRight(), muted))
			}
		}
		controls = append(controls, core.Text{Class: "text-xs " + muted, ContentFunc: func() string {
			if a.query.Get() == "" || a.target.Get() != "smart" {
				return ""
			}
			if targetLanguage(a.source.Get(), "smart", a.query.Get()) == "en" {
				return "译为英语"
			}
			return "译为中文"
		}})
	}
	controls = append(controls, box("flex-1"))
	if isChat {
		controls = append(controls, core.Dynamic(func(*core.Scope) core.Node {
			if a.generating.Get() {
				return action("停止", nil, false, a.stopChat)
			}
			return action("发送", lucide.SendHorizontal(), true, submit)
		}))
	} else {
		controls = append(controls, action("翻译", lucide.ArrowUpRight(), true, submit))
	}
	composer := box("gap-1 p-2 "+card, field, box("flex-row items-center gap-2 px-1", controls...))
	notice := core.Show{When: func() bool { return a.note.Get() != "" }, Then: func(*core.Scope) core.Node {
		return core.Text{ContentFunc: a.note.Get, Class: "text-sm text-amber-800 dark:text-amber-300"}
	}}
	foot := label("Enter 提交     Shift+Enter 换行     Ctrl+Tab 切换     Esc 收起", "text-xs "+muted)
	if isChat {
		return box("flex-1 p-4 gap-2", a.chatResults(cx), notice, composer, foot)
	}
	return box("flex-1 p-4 gap-2", composer, notice, core.Scroll{Class: "flex-1", Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node { a.resultsVersion.Get(); return a.translationResults(cx) })}}, foot)
}
func (a *application) translationResults(cx *core.Scope) core.Node {
	var rows []core.Node
	if a.found {
		e := a.entry
		query := a.submitted
		nodes := []core.Node{box("flex-row gap-2 items-center", glyph(lucide.BookOpenText(), accent), label("离线英汉词典", "text-xs font-semibold "+accent)), label(e.Word, "text-2xl font-semibold "+ink)}
		if s := strings.TrimSpace(e.Phonetic + "  " + e.POS); s != "" {
			nodes = append(nodes, label(s, "text-sm "+muted))
		}
		nodes = append(nodes, label(e.Translation, "text-base "+ink), box("flex-row gap-2", action("在线对照", nil, false, func() {
			a.startTranslation(query, true)
		}), iconButton("复制释义", lucide.Copy(), func() { cx.SetClipboard(e.Translation) })))
		rows = append(rows, box("p-4 gap-2 "+card, nodes...))
	}
	for i, r := range a.translations {
		name := r.provider
		if name == service.Baidu {
			name = "百度翻译"
		}
		if name == service.AI {
			name = "AI 翻译"
		}
		header := box("flex-row gap-2 items-center", label(name, "text-sm font-semibold "+accent), box("flex-1"))
		value := r.value
		if r.pending {
			value = "翻译中…"
		} else if r.failure != "" {
			value = r.failure
			header.Children = append(header.Children, action("重试", lucide.RotateCcw(), false, func() {
				if a.translations[i].pending {
					return
				}
				a.translations[i].pending = true
				a.resultsVersion.Update(increment)
				a.requestTranslation(a.translateCtx, i)
			}))
		} else {
			header.Children = append(header.Children, iconButton("就此提问", lucide.MessageCircle(), func() { a.askAbout(r.value) }), iconButton("复制译文", lucide.Copy(), func() { cx.SetClipboard(r.value) }))
		}
		rows = append(rows, box("p-3 gap-2 "+card, header, label(value, "text-base "+ink)))
	}
	return box("gap-2", rows...)
}
