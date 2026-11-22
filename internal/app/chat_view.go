package app

import (
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

func (a *application) chatWorkspace(cx *core.Scope) core.Node {
	copyText, feedback := copyFeedback(cx)
	follow := core.NewSignal(cx, true)
	send := func() {
		wasGenerating := a.generating.Get()
		a.send()
		if !wasGenerating && a.generating.Get() {
			follow.Set(true)
		}
	}
	sendButton := action("发送", lucide.SendHorizontal(), true, send)
	sendButton.Disabled = func() bool { return strings.TrimSpace(a.question.Get()) == "" }
	composer := box("z-10 p-3 gap-2 "+sheet,
		a.input("问答输入", a.question, "有什么想问的？", 72, send),
		box("flex-row items-center gap-2",
			action("新话题", lucide.Plus(), false, a.newConversation),
			box("flex-1"),
			core.Show{When: func() bool { return !follow.Get() }, Then: func(*core.Scope) core.Node {
				return action("查看最新消息", lucide.ArrowDown(), false, func() {
					follow.Set(true)
					a.focusEditor()
				})
			}},
			core.Dynamic(func(*core.Scope) core.Node {
				if a.generating.Get() {
					return action("停止", lucide.Square(), false, a.stopChat)
				}
				return sendButton
			}),
		),
	)
	return box("flex-1 min-h-0 px-4 pt-4 pb-3 gap-3",
		box("flex-1 min-h-0 "+sheet, a.chatResults(cx, follow, copyText)), a.panelNotice(), composer, shortcutBar("发送", feedback))
}

func (a *application) chatResults(cx *core.Scope, follow *core.Signal[bool], copyText func(string)) core.Node {
	var scroll core.Ref
	conversation := a.conversation
	cx.Effect(func() {
		a.messagesVersion.Get()
		a.partial.Get()
		if conversation != a.conversation {
			conversation = a.conversation
			follow.Set(true)
		}
		if follow.Get() {
			cx.Post(func() {
				if follow.Get() {
					scroll.ScrollToEnd(false)
				}
			})
		}
	})
	return core.Scroll{Label: "会话消息", Class: "flex-1", Ref: &scroll, OnScroll: func(event core.ScrollEvent) {
		if !event.Stale {
			follow.Set(event.ContentSize.Height-event.Size.Height-event.Offset.Y <= 24)
		}
	}, Children: []core.Node{core.Dynamic(func(*core.Scope) core.Node {
		a.messagesVersion.Get()
		nodes := []core.Node{}
		for _, m := range a.messages {
			if m.Role == "user" {
				nodes = append(nodes, box("self-end max-w-[85%] px-4 py-3 gap-1 rounded-lg bg-accent-tint",
					label("你", "text-xs font-semibold "+accent),
					core.Text{Content: m.Content, Selectable: core.SelectableOn, Class: "text-base leading-6 " + ink}))
			} else {
				nodes = append(nodes, answer(m.Content, copyText))
			}
		}
		if a.generating.Get() {
			nodes = append(nodes, box("gap-2 border-t pt-3 "+divider,
				core.Text{Content: "正在回答…", Role: core.RoleStatus, Class: "text-sm font-semibold " + accent},
				core.Text{Class: "text-base leading-6 " + ink, ContentFunc: a.partial.Get}))
		}
		if notice := a.chatErrors[a.conversation]; notice != "" {
			nodes = append(nodes, label(notice, "text-sm "+warning))
		}
		if len(nodes) == 0 {
			nodes = append(nodes, box("items-center gap-3 py-6",
				box("size-12 rounded-lg items-center justify-center bg-accent-tint", glyph(lucide.MessageCircle(), accent)),
				label("开始一段问答", "text-lg font-semibold "+ink),
				label("输入问题，也可以从译文发起提问。", "text-sm "+muted)))
		}
		return box("gap-3 pl-4 pr-5 py-4", nodes...)
	})}}
}
func answer(text string, copyText func(string)) core.Node {
	rows := []core.Node{box("flex-row items-center gap-2", label("AI 回答", "text-sm font-semibold "+accent), box("flex-1"), iconButton("复制回答", lucide.Copy(), func() { copyText(text) }))}
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
			rows = append(rows, label(s, "text-base leading-6 "+ink))
			return
		}
		caption := lang
		if caption == "" {
			caption = "代码"
		}
		rows = append(rows, box("px-3 pb-3 gap-1 "+inset, box("flex-row items-center", label(caption, "text-xs font-semibold "+muted), box("flex-1"), iconButton("复制代码", lucide.Copy(), func() { copyText(s) })), label(s, "font-mono text-sm "+ink)))
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
	v := box("pt-1 gap-2 border-t "+divider, rows...)
	v.Selectable = true
	return v
}
func (a *application) historyView(cx *core.Scope) core.Node {
	search := core.NewSignal(cx, "")
	rev := core.NewSignal(cx, 0)
	var field core.Ref
	return box("shrink px-4 pt-4 pb-3 gap-3", label("历史会话", "pl-1 text-2xl font-semibold "+ink),
		box("flex-row items-center px-3 py-2 gap-2 "+sheet,
			glyph(lucide.Search(), muted),
			ui.Input{Ref: &field, Label: "搜索历史会话", Class: "flex-1", Style: core.Style{Height: core.Pt(36)}, Placeholder: "搜索标题或对话内容", AutoFocus: true, Value: search.Get, OnChange: search.Set},
			core.Show{When: func() bool { return search.Get() != "" }, Then: func(*core.Scope) core.Node {
				return iconButton("清除搜索", lucide.X(), func() {
					search.Set("")
					field.Focus()
				})
			}},
		),
		a.panelNotice(),
		core.Scroll{Label: "会话列表", Style: core.Style{MaxHeight: core.Pt(352)}, Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
			rev.Get()
			list, err := a.db.Conversations(search.Get())
			if err != nil {
				return label(err.Error(), muted)
			}
			rows := []core.Node{}
			for _, c := range list {
				rows = append(rows, box("flex-row gap-2 items-center p-2 "+sheet, base.Button{Class: "flex-1 min-w-0 p-2 rounded-md flex-row items-center gap-3 " + buttonStates, Label: c.Title, OnPress: func(core.PressEvent) { a.switchMode("chat"); a.openConversation(c.ID) }, Children: []core.Node{glyph(lucide.MessageCircle(), accent), core.Text{Content: c.Title, MaxLines: 2, Class: "flex-1 min-w-0 text-base " + ink}, glyph(lucide.ChevronRight(), muted)}},
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
				empty := "尚无会话"
				if strings.TrimSpace(search.Get()) != "" {
					empty = "没有匹配的会话"
				}
				rows = append(rows, box("p-4 "+sheet, label(empty, "text-sm "+muted)))
			}
			return box("gap-2 pr-2", rows...)
		})}})
}
