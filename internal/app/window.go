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
	panelShadowTop           = 18
	panelShadowSide          = 22
	panelShadowBottom        = 26
	panelShadowHorizontal    = 2 * panelShadowSide
	panelShadowVertical      = panelShadowTop + panelShadowBottom
	panelBorderWidth         = 1
	translationResultsHeight = 276
)

var panelShadowEdges = core.Edges{
	Top:    panelShadowTop,
	Right:  panelShadowSide,
	Bottom: panelShadowBottom,
	Left:   panelShadowSide,
}

func label(s, class string) core.Node { return core.Text{Content: s, Class: class} }
func box(class string, children ...core.Node) core.View {
	return core.View{Class: class, Children: children}
}
func card(cx *core.Scope, class string, children ...core.Node) core.View {
	v := core.View{Class: class + " " + sheet, Children: children}
	v.StyleFunc = func() core.Style { return cardShadow(cx) }
	return v
}
func focusCard(cx *core.Scope, class string, children ...core.Node) core.View {
	v := core.View{Class: class + " " + sheet, Children: children}
	v.StyleFunc = func() core.Style { return focusCardStyle(cx) }
	return v
}
func glyph(svg *core.SVG, class string) core.Node {
	return core.Icon{SVG: svg, Class: "size-4 " + class}
}
func glyphSm(svg *core.SVG, class string) core.Node {
	return core.Icon{SVG: svg, Class: "size-3.5 " + class}
}
func pill(text, class string) core.Node {
	return box("px-2 py-0.5 rounded-full items-center justify-center "+class, label(text, "text-xs font-medium"))
}

func action(cx *core.Scope, name string, svg *core.SVG, primary bool, fn func()) base.Button {
	bg, fg := "border bg-surface "+divider+" "+buttonStates, ink
	if primary {
		bg = "bg-accent enabled:hover:bg-accent-hover enabled:active:bg-accent-pressed " + focusRing
		fg = "text-accent-text"
	}
	var content []core.Node
	if svg != nil {
		content = append(content, glyphSm(svg, fg))
	}
	content = append(content, label(name, "text-sm font-medium "+fg))
	b := base.Button{
		Label:    name,
		Class:    "flex-row gap-1.5 px-3.5 h-9 rounded-lg items-center justify-center disabled:opacity-45 " + bg,
		Style:    buttonMotion(cx, 0.975),
		OnPress:  func(core.PressEvent) { fn() },
		Children: content,
	}
	if primary {
		b.StyleFunc = func() core.Style { return primaryCTAStyle(cx) }
	}
	return b
}

func compactAction(cx *core.Scope, name string, svg *core.SVG, fn func()) base.Button {
	var content []core.Node
	if svg != nil {
		content = append(content, glyphSm(svg, muted))
	}
	content = append(content, label(name, "text-xs font-medium "+muted))
	return base.Button{
		Label:    name,
		Class:    "flex-row gap-1.5 px-2.5 h-7 rounded-md items-center justify-center border bg-surface-subtle " + subtleBorder + " " + buttonStates,
		Style:    buttonMotion(cx, 0.97),
		OnPress:  func(core.PressEvent) { fn() },
		Children: content,
	}
}

func iconButton(cx *core.Scope, name string, svg *core.SVG, fn func()) core.Node {
	return base.Button{
		Label:    name,
		Tooltip:  label(name, "text-xs"),
		Class:    "size-8 items-center justify-center rounded-lg text-text-muted " + buttonStates,
		Style:    buttonMotion(cx, 0.92),
		OnPress:  func(core.PressEvent) { fn() },
		Children: []core.Node{glyph(svg, "")},
	}
}

func compactIconButton(cx *core.Scope, name string, svg *core.SVG, fn func()) core.Node {
	return base.Button{
		Label:    name,
		Tooltip:  label(name, "text-xs"),
		Class:    "size-7 items-center justify-center rounded-md text-text-muted " + buttonStates,
		Style:    buttonMotion(cx, 0.90),
		OnPress:  func(core.PressEvent) { fn() },
		Children: []core.Node{glyphSm(svg, "")},
	}
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
		// Leave time for the 110ms fade-out so feedback finishes within 2 seconds.
		hold := 1850 * time.Millisecond
		if ui.ReduceMotion(cx) {
			hold = 2 * time.Second
		}
		stop = cx.AfterFunc(hold, func() { copied.Set(false) })
	}
	return copyText, presencePop(copied.Get, func(*core.Scope) core.Node {
		return box("flex-row items-center gap-1 px-2 py-0.5 rounded-full bg-accent-tint",
			glyphSm(lucide.Check(), accent),
			core.Text{Role: core.RoleStatus, Content: "已复制", Class: "text-xs font-medium " + accent},
		)
	})
}

func modeButton(cx *core.Scope, name string, icon *core.SVG, selected func() bool, activate func()) core.Node {
	return base.Button{
		Label:    name,
		Selected: selected,
		OnPress:  func(core.PressEvent) { activate() },
		Class:    "flex-row gap-1.5 h-7 px-3 rounded-md items-center justify-center text-text-muted " + buttonStates + " " + selectedFill + " aria-selected:text-accent",
		Style:    buttonMotion(cx, 0.97),
		Children: []core.Node{glyphSm(icon, ""), label(name, "text-xs font-semibold")},
	}
}

func brandMark(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	v := box("size-7 rounded-lg items-center justify-center bg-accent", glyphSm(lucide.Terminal(), "text-accent-text"))
	v.StyleFunc = func() core.Style {
		alpha := float32(0.28)
		if cx.Dark() {
			alpha = 0.32
		}
		return core.Style{
			BorderCurve: core.BorderCurveContinuous,
			Shadows: []core.BoxShadow{{
				Offset: core.Point{Y: 1},
				Blur:   3,
				Color:  colors.Accent().Faded(alpha),
			}},
		}
	}
	return v
}

func searchBar(cx *core.Scope, ref *core.Ref, fieldLabel, placeholder string, value *core.Signal[string]) core.Node {
	colors := ui.ColorsOf(cx)
	return focusCard(cx, "flex-row items-center px-3 py-1 gap-2",
		glyphSm(lucide.Search(), faint),
		ui.Input{
			Ref:         ref,
			Label:       fieldLabel,
			Placeholder: placeholder,
			AutoFocus:   true,
			Class:       "flex-1",
			Value:       value.Get,
			OnChange:    value.Set,
			Style: core.Style{
				Height:      core.Pt(34),
				FontSize:    14,
				Padding:     core.Symmetric(0, 2),
				BorderStyle: core.BorderNone,
			},
			StyleFunc: func() core.Style {
				return core.Style{Background: colors.Surface()}
			},
		},
		core.Show{When: func() bool { return value.Get() != "" }, Then: func(cx *core.Scope) core.Node {
			return popReveal(cx, 0, compactIconButton(cx, "清除搜索", lucide.X(), func() {
				value.Set("")
				ref.Focus()
			}))
		}},
	)
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
	window := core.View{
		StyleFunc: func() core.Style {
			if desktop.Transparent(cx) {
				return core.Style{
					Width:   core.Pt(panelWidth + panelShadowHorizontal),
					Padding: panelShadowEdges,
				}
			}
			return core.Style{Width: core.Pt(panelWidth)}
		},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				page, mode := a.page.Get(), a.mode.Get()
				transparent := desktop.Transparent(cx)
				header := box("flex-row items-center gap-1.5 px-4 py-2.5 bg-surface border-b "+divider,
					brandMark(cx),
					label("Go Ask", "ml-1.5 mr-4 font-semibold text-base "+ink),
				)
				if page == "work" {
					header.Children = append(header.Children,
						box("flex-row items-center gap-0.5 p-1 rounded-lg bg-inset border "+subtleBorder,
							modeButton(cx, "翻译", lucide.Languages(), func() bool { return a.mode.Get() == "translate" }, func() { a.switchMode("translate") }),
							modeButton(cx, "问答", lucide.MessageCircle(), func() bool { return a.mode.Get() == "chat" }, func() { a.switchMode("chat") }),
						),
					)
				} else {
					back := compactAction(cx, "返回", lucide.ChevronLeft(), func() { a.switchMode(a.mode.Get()) })
					header.Children = append(header.Children,
						back,
						label("历史会话", "ml-2 text-sm font-semibold "+ink),
					)
				}
				header.Children = append(header.Children, core.View{Class: "flex-1 h-8", Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOn}})
				if page == "work" {
					header.Children = append(header.Children,
						iconButton(cx, "历史会话", lucide.Clock3(), func() {
							a.note.Set("")
							a.page.Set("history")
						}),
						iconButton(cx, "设置", lucide.Settings2(), a.host.openSettings),
						box("mx-1 h-4 w-px bg-border"),
					)
				}
				pinIcon := core.Icon{
					SVG:   lucide.Pin(),
					Class: "size-4",
					StyleFunc: func() core.Style {
						if ui.ReduceMotion(cx) {
							return core.Style{}
						}
						deg := float32(0)
						if pinned.Get() {
							deg = -22
						}
						return core.Style{
							Transforms: rotateSteps(deg),
							Transition: core.Transition{
								Duration:   motionNormal,
								Easing:     core.EaseOut,
								Properties: core.TransitionTransform,
							},
						}
					},
				}
				header.Children = append(header.Children,
					core.Show{When: func() bool { return !canPin() }, Then: func(*core.Scope) core.Node {
						return label(pinUnavailable, "mr-1 text-xs "+muted)
					}},
					base.Button{
						LabelFunc: pinLabel,
						Tooltip:   core.Text{Class: "text-xs", ContentFunc: pinLabel},
						Disabled:  func() bool { return !canPin() },
						Selected:  pinned.Get,
						Class:     "size-8 items-center justify-center rounded-lg text-text-muted disabled:opacity-45 " + buttonStates + " " + selectedFill + " aria-selected:text-accent",
						Style:     buttonMotion(cx, 0.92),
						OnPress:   func(core.PressEvent) { a.host.togglePanelPin() },
						Children:  []core.Node{pinIcon},
					},
					iconButton(cx, "收起窗口", lucide.Minus(), a.host.hide),
				)
				var body core.Node
				switch {
				case page == "history":
					body = enterReveal(cx, 0, 6, a.historyView(cx))
				case mode == "chat":
					body = enterRevealFlex(cx, 6, a.chatWorkspace(cx))
				default:
					body = enterReveal(cx, 0, 6, a.translationWorkspace(cx))
				}
				inner := box("shrink overflow-hidden", header, body)
				if transparent {
					inner.Style.CornerRadius = 15
					inner.Style.BorderCurve = core.BorderCurveContinuous
				}
				if page == "work" && mode == "chat" {
					// Keep the composer in one place as a streamed reply grows.
					inner.Style.Height = core.Pt(panelMaxHeight - 2*panelBorderWidth)
				}
				frame := core.View{
					StyleFunc: func() core.Style { return floatingPanelStyle(cx, transparent) },
					Children:  []core.Node{inner},
				}
				return frame
			}),
		},
	}
	return window
}

func (a *application) composerInput(cx *core.Scope, name string, draft *core.Signal[string], placeholder string, height core.Points, submit func()) ui.Input {
	colors := ui.ColorsOf(cx)
	return ui.Input{
		Ref:         &a.editor,
		Label:       name,
		AutoFocus:   true,
		Value:       draft.Get,
		OnChange:    draft.Set,
		Placeholder: placeholder,
		Multiline:   true,
		Style: core.Style{
			Height:      core.Pt(height),
			FontSize:    15,
			LineHeight:  22,
			Padding:     core.Edges{Top: 12, Right: 14, Bottom: 8, Left: 14},
			BorderStyle: core.BorderNone,
		},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Surface()}
		},
		SubmitBehavior: core.SubmitBehaviorSubmit,
		OnSubmit:       func(text string) { draft.Set(text); submit() },
	}
}

func (a *application) panelNotice() core.Node {
	return core.Show{When: func() bool { return a.note.Get() != "" }, Then: func(cx *core.Scope) core.Node {
		return popReveal(cx, -4, box("flex-row items-center gap-2 px-3.5 py-2.5 "+noticeBox,
			glyphSm(lucide.CircleAlert(), warning),
			core.Text{Role: core.RoleStatus, ContentFunc: a.note.Get, Class: "flex-1 min-w-0 text-sm " + warning},
		))
	}}
}

func shortcutBar(verb string, feedback core.Node) core.Node {
	footer := box("flex-row flex-wrap gap-x-4 gap-y-1 items-center px-1 min-h-6")
	for _, shortcut := range [][2]string{{"Enter", verb}, {"Shift+Enter", "换行"}, {"Ctrl+Tab", "切换"}, {"Esc", "收起"}} {
		footer.Children = append(footer.Children, box("flex-row items-center gap-1.5",
			box("h-5 px-1.5 rounded border bg-surface justify-center "+divider, label(shortcut[0], "font-mono text-xs "+muted)),
			label(shortcut[1], "text-xs "+muted),
		))
	}
	footer.Children = append(footer.Children, core.View{Class: "flex-1"}, feedback)
	return footer
}

func (a *application) translationWorkspace(cx *core.Scope) core.Node {
	copyText, feedback := copyFeedback(cx)
	translate := action(cx, "翻译", lucide.ArrowRight(), true, a.translate)
	translate.TestID = "translation-submit"
	translate.Disabled = func() bool { return strings.TrimSpace(a.query.Get()) == "" }
	language := func(name, first, title string, value *core.Signal[string]) core.Node {
		return ui.Select[string]{
			Label: name,
			Options: []ui.Option[string]{
				{Value: first, Label: title}, {Value: "zh", Label: "中文"}, {Value: "en", Label: "英语"},
				{Value: "ja", Label: "日语"}, {Value: "fr", Label: "法语"}, {Value: "de", Label: "德语"},
			},
			Value:    value.Get,
			OnChange: value.Set,
			Style:    core.Style{Width: core.Pt(118), Height: core.Pt(32)},
		}
	}
	swapTurns := core.NewSignal(cx, 0)
	swapLanguages := func() {
		src, dst := a.source.Get(), a.target.Get()
		nextSrc, nextDst := dst, src
		switch {
		case src == "auto" && dst == "smart":
			if targetLanguage("auto", "smart", a.query.Get()) == "en" {
				nextSrc, nextDst = "en", "zh"
			} else {
				nextSrc, nextDst = "zh", "en"
			}
		case src == "auto":
			nextSrc = dst
			if dst == "zh" {
				nextDst = "en"
			} else {
				nextDst = "zh"
			}
		case dst == "smart":
			if src == "zh" {
				nextSrc, nextDst = "en", "zh"
			} else {
				nextSrc, nextDst = "zh", src
			}
		}
		cx.Batch(func() {
			a.source.Set(nextSrc)
			a.target.Set(nextDst)
			swapTurns.Update(increment)
		})
	}
	swapIcon := core.Icon{
		SVG:   lucide.ArrowLeftRight(),
		Class: "size-3.5 text-text-muted",
		StyleFunc: func() core.Style {
			if ui.ReduceMotion(cx) {
				return core.Style{}
			}
			return core.Style{
				Transforms: rotateSteps(float32((swapTurns.Get() % 2) * 180)),
				Transition: core.Transition{
					Duration:   motionEnter,
					Easing:     core.EaseOut,
					Properties: core.TransitionTransform,
				},
			}
		},
	}
	swapButton := base.Button{
		Label:    "交换语言",
		Tooltip:  label("交换源语言与目标语言", "text-xs"),
		Class:    "size-7 items-center justify-center rounded-md text-text-muted " + buttonStates,
		Style:    buttonMotion(cx, 0.90),
		OnPress:  func(core.PressEvent) { swapLanguages() },
		Children: []core.Node{swapIcon},
	}
	// Keep the editor's hit area above scrolled result rows.
	composer := focusCard(cx, "z-10 overflow-hidden",
		a.composerInput(cx, "翻译原文", a.query, "输入单词或粘贴一段文字…", 80, a.translate),
		box("flex-row items-center gap-2 px-3 py-2 bg-surface-subtle border-t "+subtleBorder,
			language("源语言", "auto", "自动识别", a.source),
			swapButton,
			language("目标语言", "smart", "中英互译", a.target),
			core.Show{
				When: func() bool {
					return strings.TrimSpace(a.query.Get()) != "" && a.target.Get() == "smart"
				},
				Then: func(cx *core.Scope) core.Node {
					return popReveal(cx, 0, box("ml-1 px-2 py-0.5 rounded-full bg-inset",
						core.Text{Class: "text-xs font-medium " + muted, ContentFunc: func() string {
							if targetLanguage(a.source.Get(), "smart", a.query.Get()) == "en" {
								return "译为英语"
							}
							return "译为中文"
						}},
					))
				},
			},
			box("flex-1"),
			core.Show{
				When: func() bool { return a.query.Get() != "" },
				Then: func(cx *core.Scope) core.Node {
					return popReveal(cx, 0, compactIconButton(cx, "清空原文", lucide.X(), func() {
						a.query.Set("")
						a.clearTranslation()
						a.focusEditor()
					}))
				},
			},
			translate,
		),
	)
	var resultsScroll core.Ref
	queryID := a.translationID
	animatedID := a.translationID
	settled := make(map[string]bool)
	cx.Effect(func() {
		a.resultsVersion.Get()
		if queryID != a.translationID {
			queryID = a.translationID
			cx.Post(func() { resultsScroll.ScrollTo(core.Point{}, false) })
		}
	})
	return box("shrink px-4 pt-3.5 pb-3 gap-3", composer, a.panelNotice(),
		core.Scroll{
			Label: "翻译结果", Ref: &resultsScroll,
			// Keep the editor and keyboard hints outside the scroll viewport.
			Style: core.Style{MaxHeight: core.Pt(translationResultsHeight)},
			Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
				a.resultsVersion.Get()
				freshQuery := animatedID != a.translationID
				if freshQuery {
					animatedID = a.translationID
					clear(settled)
				}
				return a.translationResults(cx, copyText, freshQuery, settled)
			})},
		}, shortcutBar("翻译", feedback))
}

func (a *application) translationResults(cx *core.Scope, copyText func(string), freshQuery bool, settled map[string]bool) core.Node {
	var rows []core.Node
	if a.found {
		e := a.entry
		query := a.submitted
		var meta []core.Node
		if p := strings.TrimSpace(e.Phonetic); p != "" {
			meta = append(meta, box("px-2 py-0.5 rounded-md bg-inset", label(p, "font-mono text-xs "+muted)))
		}
		if pos := strings.TrimSpace(e.POS); pos != "" {
			meta = append(meta, pill(pos, "border border-accent-border bg-accent-tint text-accent"))
		}
		headword := box("gap-1.5",
			core.Text{
				Content: e.Word,
				Class:   "text-3xl font-semibold " + ink,
				Style:   core.Style{FontFamily: "Georgia, serif"},
			},
		)
		if len(meta) > 0 {
			headword.Children = append(headword.Children, box("flex-row flex-wrap items-center gap-2", meta...))
		}
		var dictNode core.Node = card(cx, "p-4 gap-3",
			box("flex-row items-center justify-between gap-2",
				box("flex-row items-center gap-1.5 px-2.5 py-1 rounded-full bg-accent-tint",
					glyphSm(lucide.BookOpen(), accent),
					label("离线英汉词典", "text-xs font-semibold "+accent),
				),
				box("flex-row items-center gap-1.5",
					compactAction(cx, "在线对照", lucide.Globe(), func() {
						a.startTranslation(query, true)
					}),
					copyIconButton(cx, "复制释义", func() { copyText(e.Translation) }),
				),
			),
			headword,
			box("border-t pt-3 "+subtleBorder,
				core.Text{
					Content:    strings.TrimSpace(e.Translation),
					Selectable: core.SelectableOn,
					Class:      "text-base " + ink,
					Style:      core.Style{LineHeight: 24},
				},
			),
		)
		if freshQuery {
			dictNode = enterReveal(cx, 0, 8, dictNode)
		}
		rows = append(rows, dictNode)
	}
	for i, r := range a.translations {
		provider, _ := service.ProviderByID(r.provider)
		var actions []core.Node
		var body core.Node
		if r.pending {
			settled[r.provider] = false
			body = box("flex-row items-center gap-2 py-1",
				ui.Spinner{Size: 14, Label: "翻译中"},
				label("翻译中…", "text-sm "+muted),
			)
		} else if r.failure != "" {
			body = box("flex-row items-start gap-2",
				glyphSm(lucide.CircleAlert(), warning),
				core.Text{Content: r.failure, Selectable: core.SelectableOn, Class: "flex-1 min-w-0 text-sm " + warning},
			)
			if !settled[r.provider] {
				settled[r.provider] = true
				body = enterReveal(cx, 0, 4, body)
			}
			actions = append(actions, compactAction(cx, "重试", lucide.RotateCcw(), func() {
				if a.translations[i].pending {
					return
				}
				settled[r.provider] = false
				a.translations[i].pending = true
				a.resultsVersion.Update(increment)
				a.requestTranslation(a.translateCtx, i)
			}))
		} else {
			body = core.Text{
				Content:    r.value,
				Selectable: core.SelectableOn,
				Class:      "text-base " + ink,
				Style:      core.Style{LineHeight: 24},
			}
			if !settled[r.provider] {
				settled[r.provider] = true
				body = enterReveal(cx, 0, 4, body)
			}
			actions = append(actions,
				compactAction(cx, "就此提问", lucide.MessageCircle(), func() { a.askAbout(r.value) }),
				copyIconButton(cx, "复制译文", func() { copyText(r.value) }),
			)
		}
		providerIcon := lucide.Globe()
		if provider.AI {
			providerIcon = lucide.Cpu()
		}
		var cardNode core.Node = card(cx, "overflow-hidden",
			box("flex-row items-center justify-between gap-2 px-4 py-2 bg-surface-subtle border-b "+subtleBorder,
				box("flex-row items-center gap-2 flex-1 min-w-0",
					glyphSm(providerIcon, accent),
					label(strings.TrimSuffix(provider.Name, "（内置）"), "flex-1 min-w-0 text-xs font-semibold "+ink),
				),
				box("flex-row items-center gap-1.5", actions...),
			),
			box("px-4 py-3", body),
		)
		if freshQuery {
			step := min(i, 3)
			cardNode = enterReveal(cx, time.Duration(step)*motionStagger, 8, cardNode)
		}
		rows = append(rows, cardNode)
	}
	if len(rows) == 0 {
		if a.submitted.text != "" {
			return core.View{}
		}
		return box("flex-row items-center gap-3.5 px-4 py-4 "+cardSubtle,
			box("size-10 rounded-xl border border-accent-border bg-accent-tint items-center justify-center shrink-0", glyph(lucide.Languages(), accent)),
			box("flex-1 gap-1.5",
				box("gap-0.5",
					label("输入文字后按 Enter 开始翻译", "text-sm font-semibold "+ink),
					label("单词优先查本地词典，句段自动并发请求已启用的在线服务。", "text-xs "+muted),
				),
				box("flex-row flex-wrap items-center gap-1.5",
					pill("离线词典", "bg-inset text-text-muted"),
					pill("多源对照", "bg-inset text-text-muted"),
					pill("划词即译", "bg-inset text-text-muted"),
				),
			),
		)
	}
	if !a.found {
		rows = append([]core.Node{box("flex-row items-center gap-2 px-1 py-0.5",
			pill("原文", "bg-accent-tint text-accent"),
			core.Text{Content: a.submitted.text, MaxLines: 1, Class: "flex-1 min-w-0 text-xs " + muted},
		)}, rows...)
	}
	return box("gap-2.5", rows...)
}
