package app

import (
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
)

func (a *application) chatResults(cx *core.Scope) core.Node {
	var scroll core.Ref
	cx.Effect(func() { a.messagesVersion.Get(); a.partial.Get(); cx.Post(func() { scroll.ScrollToEnd(false) }) })
	return core.Scroll{Class: "flex-1", Ref: &scroll, Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
		a.messagesVersion.Get()
		nodes := []core.Node{}
		for _, m := range a.messages {
			if m.Role == "user" {
				nodes = append(nodes, box("self-end max-w-full p-3 rounded-lg bg-indigo-100 dark:bg-indigo-950", label(m.Content, "text-base "+ink)))
			} else {
				nodes = append(nodes, a.answer(cx, m.Content))
			}
		}
		if a.generating.Get() {
			nodes = append(nodes, core.Text{Class: "p-3 text-base " + ink, ContentFunc: func() string {
				if s := a.partial.Get(); s != "" {
					return s
				}
				return "正在回答…"
			}})
		}
		if notice := a.chatErrors[a.conversation]; notice != "" {
			nodes = append(nodes, label(notice, "text-sm text-amber-800 dark:text-amber-300"))
		}
		if len(nodes) == 0 {
			nodes = append(nodes, box("p-3 gap-2 self-start", glyph(lucide.MessageCircle(), accent), label("快速问答", "text-lg font-semibold "+ink), label("例如：Linux 怎么查看占用 8080 端口的进程？", "text-sm "+muted)))
		}
		return box("gap-2 pb-2", nodes...)
	})}}
}
func (a *application) answer(cx *core.Scope, text string) core.Node {
	rows := []core.Node{box("flex-row items-center gap-2", glyph(lucide.Sparkles(), accent), label("AI 回答", "text-sm font-semibold "+accent), box("flex-1"), iconButton("复制回答", lucide.Copy(), func() { cx.SetClipboard(text) }))}
	// Only fenced blocks need a distinct action. Keep explanatory text selectable.
	code := false
	var lines []string
	lang := ""
	flush := func() {
		s := strings.Trim(strings.Join(lines, "\n"), "\n")
		lines = nil
		if s == "" {
			return
		}
		if !code {
			rows = append(rows, label(s, "text-base "+ink))
			return
		}
		caption := lang
		if caption == "" {
			caption = "代码"
		}
		rows = append(rows, box("p-3 gap-1 rounded-lg bg-slate-900", box("flex-row items-center", label(caption, "text-xs text-slate-300"), box("flex-1"), action("复制代码", lucide.Copy(), false, func() { cx.SetClipboard(s) })), label(s, "font-mono text-sm text-slate-100")))
	}
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			flush()
			code = !code
			lang = strings.TrimSpace(strings.TrimPrefix(trim, "```"))
			continue
		}
		lines = append(lines, line)
	}
	flush()
	v := box("p-3 gap-2 "+card, rows...)
	v.Selectable = true
	return v
}
func (a *application) historyView(cx *core.Scope) core.Node {
	search := core.NewSignal(cx, "")
	rev := core.NewSignal(cx, 0)
	return box("flex-1 p-4 gap-3", label("历史会话", "text-xl font-semibold "+ink),
		core.Input{Class: "w-full h-10 px-3 " + card + " " + ink, Placeholder: "搜索标题或对话内容", AutoFocus: true, Value: search.Get, OnChange: search.Set},
		core.Scroll{Class: "flex-1", Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
			rev.Get()
			list, err := a.db.Conversations(search.Get())
			if err != nil {
				return label(err.Error(), muted)
			}
			rows := []core.Node{}
			for _, c := range list {
				rows = append(rows, box("flex-row gap-2 items-center p-2 "+card, core.View{Class: "flex-1 p-2", Role: core.RoleButton, Label: c.Title, OnPress: func(core.PressEvent) { a.switchMode("chat"); a.openConversation(c.ID) }, Children: []core.Node{core.Text{Content: c.Title, MaxLines: 2, Class: "text-base " + ink}}},
					iconButton("删除会话", lucide.Trash(), func() {
						cx.ShowAlert(core.AlertConfig{Title: "删除这段会话？", Message: c.Title, Cancelable: true, Actions: []core.AlertAction{{Label: "取消"}, {Label: "删除"}}}, func(n int) {
							if n != 1 {
								return
							}
							if c.ID == a.conversation {
								a.stopChat()
							}
							if err := a.db.DeleteConversation(c.ID); err != nil {
								a.note.Set(err.Error())
								return
							}
							delete(a.chatErrors, c.ID)
							if c.ID == a.conversation {
								a.conversation = 0
								a.messages = nil
								a.question.Set("")
								a.messagesVersion.Update(increment)
							}
							rev.Update(increment)
						})
					})))
			}
			if len(rows) == 0 {
				rows = append(rows, label("没有匹配的会话", "p-3 text-sm "+muted))
			}
			return box("gap-2", rows...)
		})}})
}
