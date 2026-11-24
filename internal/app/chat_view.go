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
	sendButton := action(cx, "发送", lucide.SendHorizontal(), true, send)
	sendButton.Disabled = func() bool { return strings.TrimSpace(a.question.Get()) == "" }
	composer := card(cx, "z-10 overflow-hidden",
		a.composerInput(cx, "问答输入", a.question, "有什么想问的？", 68, send),
		box("flex-row items-center gap-2 px-3 py-2 bg-surface-subtle border-t "+subtleBorder,
			compactAction(cx, "新话题", lucide.Plus(), a.newConversation),
			box("flex-1"),
			core.Show{When: func() bool { return !follow.Get() }, Then: func(cx *core.Scope) core.Node {
				return popReveal(cx, 6, compactAction(cx, "查看最新消息", lucide.ArrowDown(), func() {
					follow.Set(true)
					a.focusEditor()
				}))
			}},
			core.Dynamic(func(cx *core.Scope) core.Node {
				if a.generating.Get() {
					return popReveal(cx, 0, action(cx, "停止", lucide.Square(), false, a.stopChat))
				}
				return sendButton
			}),
		),
	)
	return box("flex-1 min-h-0 px-4 pt-3.5 pb-3 gap-3",
		card(cx, "flex-1 min-h-0 overflow-hidden", a.chatResults(cx, follow, copyText)),
		a.panelNotice(),
		composer,
		shortcutBar("发送", feedback),
	)
}

func (a *application) chatResults(cx *core.Scope, follow *core.Signal[bool], copyText func(string)) core.Node {
	var scroll core.Ref
	conversation := a.conversation
	seenConv := a.conversation
	seenCount := len(a.messages)
	wasGenerating := a.generating.GetUntracked()
	laidOut := false
	cx.Effect(func() {
		a.messagesVersion.Get()
		a.partial.Get()
		if conversation != a.conversation {
			conversation = a.conversation
			follow.Set(true)
		}
		if follow.Get() && laidOut {
			cx.Post(func() {
				if follow.Get() {
					scroll.ScrollToEnd(false)
				}
			})
		}
	})
	return core.Scroll{
		Label: "会话消息",
		Class: "flex-1",
		Ref:   &scroll,
		OnLayout: func(b core.Bounds) {
			if b.Size.Height <= 0 {
				return
			}
			if !laidOut {
				laidOut = true
				if follow.GetUntracked() {
					scroll.ScrollToEnd(false)
				}
			}
		},
		OnScroll: func(event core.ScrollEvent) {
			if event.Stale || !laidOut {
				return
			}
			follow.Set(event.ContentSize.Height-event.Size.Height-event.Offset.Y <= 24)
		},
		Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
			a.messagesVersion.Get()
			if seenConv != a.conversation {
				seenConv = a.conversation
				seenCount = len(a.messages)
			}
			prevCount := seenCount
			if prevCount > len(a.messages) {
				prevCount = len(a.messages)
			}
			seenCount = len(a.messages)

			nodes := []core.Node{}
			for i, m := range a.messages {
				var msgNode core.Node
				if m.Role == "user" {
					msgNode = box("self-end max-w-[82%] px-4 py-2.5 rounded-2xl border border-accent-border bg-accent-tint",
						core.Text{
							Content:    m.Content,
							Selectable: core.SelectableOn,
							Class:      "text-base " + ink,
							Style:      core.Style{LineHeight: 23},
						},
					)
				} else {
					msgNode = answer(cx, m.Content, i > 0, copyText)
				}
				if i >= prevCount {
					msgNode = enterReveal(cx, 0, 6, msgNode)
				}
				nodes = append(nodes, msgNode)
			}
			generating := a.generating.Get()
			if generating {
				streaming := box("gap-2",
					box("flex-row items-center gap-2",
						box("size-6 rounded-md bg-accent-tint items-center justify-center", glyphSm(lucide.Terminal(), accent)),
						ui.Spinner{Size: 13, Label: "正在回答"},
						core.Text{Content: "正在回答…", Role: core.RoleStatus, Class: "text-xs font-semibold " + accent},
					),
					core.Text{
						Class:       "pl-8 text-base " + ink,
						Style:       core.Style{LineHeight: 24},
						ContentFunc: a.partial.Get,
					},
				)
				if len(nodes) > 0 {
					streaming.Class = "pt-3 border-t " + subtleBorder + " gap-2"
				}
				var streamNode core.Node = streaming
				if !wasGenerating {
					streamNode = enterReveal(cx, 0, 6, streamNode)
				}
				nodes = append(nodes, streamNode)
			}
			wasGenerating = generating
			if notice := a.chatErrors[a.conversation]; notice != "" {
				nodes = append(nodes, popReveal(cx, -4, box("flex-row items-start gap-2 px-3.5 py-2.5 "+noticeBox,
					glyphSm(lucide.CircleAlert(), warning),
					label(notice, "flex-1 min-w-0 text-sm "+warning),
				)))
			}
			if len(nodes) == 0 {
				nodes = append(nodes, box("items-center justify-center gap-3 py-10 px-6",
					box("size-12 rounded-2xl border border-accent-border bg-accent-tint items-center justify-center",
						glyph(lucide.Terminal(), accent),
					),
					box("items-center gap-1",
						label("开始一段问答", "text-lg font-semibold "+ink),
						label("直接输入问题，或在翻译结果中点击「就此提问」追问上下文。", "text-sm "+muted),
					),
				))
			}
			return box("gap-4 pl-5 pr-5 py-4", nodes...)
		})},
	}
}

func answer(cx *core.Scope, text string, separated bool, copyText func(string)) core.Node {
	rows := []core.Node{
		box("flex-row items-center gap-2",
			box("size-6 rounded-md bg-accent-tint items-center justify-center", glyphSm(lucide.Terminal(), accent)),
			label("AI 回答", "text-xs font-semibold "+accent),
			box("flex-1"),
			copyIconButton(cx, "复制回答", func() { copyText(text) }),
		),
	}
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
			rows = append(rows, core.Text{
				Content: s,
				Class:   "pl-8 text-base " + ink,
				Style:   core.Style{LineHeight: 24},
			})
			return
		}
		caption := lang
		if caption == "" {
			caption = "代码"
		}
		rows = append(rows, box("ml-8 rounded-lg border overflow-hidden bg-code-bg "+subtleBorder,
			box("flex-row items-center px-3 py-1.5 bg-code-header border-b "+subtleBorder,
				label(caption, "font-mono text-xs font-medium "+muted),
				box("flex-1"),
				copyIconButton(cx, "复制代码", func() { copyText(s) }),
			),
			box("px-3.5 py-2.5",
				core.Text{
					Content: s,
					Class:   "font-mono text-xs " + ink,
					Style:   core.Style{LineHeight: 19},
				},
			),
		))
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
	cls := "gap-2"
	if separated {
		cls = "pt-3 border-t " + subtleBorder + " gap-2"
	}
	v := box(cls, rows...)
	v.Selectable = true
	return v
}

func (a *application) historyView(cx *core.Scope) core.Node {
	search := core.NewSignal(cx, "")
	rev := core.NewSignal(cx, 0)
	var field core.Ref
	return box("shrink px-4 pt-3.5 pb-3 gap-3",
		searchBar(cx, &field, "搜索历史会话", "搜索标题或对话内容", search),
		a.panelNotice(),
		core.Scroll{
			Label: "会话列表",
			Style: core.Style{MaxHeight: core.Pt(380)},
			Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
				rev.Get()
				list, err := a.db.Conversations(search.Get())
				if err != nil {
					return label(err.Error(), muted)
				}
				rows := []core.Node{}
				for _, c := range list {
					rowCard := card(cx, "flex-row gap-1.5 items-center pl-2 pr-2.5 py-1.5",
						base.Button{
							Class: "flex-1 min-w-0 px-2.5 py-2 rounded-lg flex-row items-center gap-3 " + buttonStates,
							Style: buttonMotion(cx, 0.988),
							Label: c.Title,
							OnPress: func(core.PressEvent) {
								a.switchMode("chat")
								a.openConversation(c.ID)
							},
							Children: []core.Node{
								box("size-7 rounded-md bg-accent-tint items-center justify-center shrink-0",
									glyphSm(lucide.MessageCircle(), accent),
								),
								core.Text{Content: c.Title, MaxLines: 2, Class: "flex-1 min-w-0 text-sm font-medium " + ink},
								glyphSm(lucide.ChevronRight(), faint),
							},
						},
						compactIconButton(cx, "删除会话", lucide.Trash(), func() {
							cx.ShowAlert(core.AlertConfig{
								Title:      "删除这段会话？",
								Message:    c.Title,
								Cancelable: true,
								Actions:    []core.AlertAction{{Label: "取消"}, {Label: "删除"}},
							}, func(n int) {
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
						}),
					)
					rowCard.StyleFunc = func() core.Style { return rowCardMotion(cx) }
					rows = append(rows, rowCard)
				}
				if len(rows) == 0 {
					empty, hint := "尚无历史会话", "在问答模式下发起对话后，记录会自动保存在本地。"
					if strings.TrimSpace(search.Get()) != "" {
						empty, hint = "没有匹配的会话", "试试更简短的关键词，或清除搜索条件。"
					}
					rows = append(rows, box("items-center justify-center gap-2 py-8 px-4 "+cardSubtle,
						box("size-10 rounded-xl bg-inset items-center justify-center", glyph(lucide.Clock3(), faint)),
						label(empty, "text-sm font-semibold "+ink),
						label(hint, "text-xs "+muted),
					))
				}
				return box("gap-2 pr-1", rows...)
			})},
		},
	)
}
