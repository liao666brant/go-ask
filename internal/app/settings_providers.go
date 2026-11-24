package app

import (
	"strings"

	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

type providerDraft struct {
	pref    providerPreference
	secrets map[string]string
}

type providerPage struct {
	selected  *core.Signal[*service.Provider]
	directory *core.Signal[bool]
	search    *core.Signal[string]
}

func (a *application) providerSettingsView(cx *core.Scope, status *core.Signal[string]) (func(bool) core.Node, func() bool, func()) {
	newPage := func() providerPage {
		return providerPage{
			selected:  core.NewSignal[*service.Provider](cx, nil),
			directory: core.NewSignal(cx, false),
			search:    core.NewSignal(cx, ""),
		}
	}
	translation, ai := newPage(), newPage()
	drafts := make(map[string]*providerDraft)
	chat := core.NewSignal(cx, a.chatProviderID())
	modalOpen := func() bool {
		return translation.directory.GetUntracked() || ai.directory.GetUntracked()
	}
	closeModal := func() {
		translation.directory.Set(false)
		ai.directory.Set(false)
	}

	view := func(isAI bool) core.Node {
		page := translation
		if isAI {
			page = ai
		}
		pick := func(provider *service.Provider) {
			page.directory.Set(false)
			page.selected.Set(provider)
			status.Set("")
		}
		content := core.Dynamic(func(cx *core.Scope) core.Node {
			if provider := page.selected.Get(); provider != nil {
				if provider.BuiltIn {
					return enterReveal(cx, 0, 6, a.builtInProviderForm(cx, provider, status, func() { pick(nil) }))
				}
				draft := drafts[provider.ID]
				if draft == nil {
					pref, secrets := a.providerSettings(provider.ID)
					draft = &providerDraft{pref: pref, secrets: secrets}
					drafts[provider.ID] = draft
				}
				return enterReveal(cx, 0, 6, a.providerForm(cx, provider, draft, chat, status, func() { pick(nil) }))
			}

			add := action(cx, "添加服务", lucide.Plus(), true, func() {
				page.search.Set("")
				page.directory.Set(true)
				status.Set("")
			})
			var items []core.Node
			for i := range service.Providers {
				provider := &service.Providers[i]
				if provider.AI != isAI || !a.hasProvider(provider.ID) {
					continue
				}
				pref, _ := a.providerSettings(provider.ID)
				var badges []core.Node
				if pref.Translate {
					badges = append(badges, pill("已启用翻译", "bg-accent-tint text-accent"))
				} else {
					badges = append(badges, pill("翻译已停用", "bg-inset text-text-muted"))
				}
				if isAI && chat.Get() == provider.ID {
					badges = append(badges, pill("当前问答提供商", "border border-accent-border bg-accent-tint text-accent"))
				}
				items = append(items, providerRow(cx, provider.Name, provider.Name, isAI, badges, func() { pick(provider) }))
			}
			if len(items) == 0 {
				hint := "添加免配置来源，即可开始在线翻译。"
				emptyIcon := lucide.Languages()
				if isAI {
					hint = "添加提供商并选择用于问答。"
					emptyIcon = lucide.Cpu()
				}
				items = append(items, box("items-center justify-center gap-2 py-8 px-4 "+cardSubtle,
					box("size-10 rounded-xl bg-accent-tint items-center justify-center", glyph(emptyIcon, accent)),
					label("尚未添加服务", "text-sm font-semibold "+ink),
					label(hint, "text-xs "+muted),
				))
			}
			sectionTitle := "已添加的翻译来源"
			if isAI {
				sectionTitle = "已添加的 AI 服务"
			}
			return box("pb-4 gap-3",
				box("flex-row items-center justify-between gap-2",
					label(sectionTitle, "pl-1 text-xs font-semibold "+muted),
					add,
				),
				box("gap-2", items...),
			)
		})
		return core.Group([]core.Node{
			content,
			a.providerDirectoryModal(cx, page, isAI, pick),
		})
	}
	return view, modalOpen, closeModal
}

func (a *application) providerDirectoryModal(cx *core.Scope, page providerPage, isAI bool, pick func(*service.Provider)) core.Node {
	colors := ui.ColorsOf(cx)
	title := "添加翻译服务"
	subtitle := "搜索并选择要接入的在线翻译来源"
	headerIcon := lucide.Languages()
	if isAI {
		title = "添加 AI 提供商"
		subtitle = "搜索并选择要接入的 AI 模型服务"
		headerIcon = lucide.Cpu()
	}
	closeDialog := func() { page.directory.Set(false) }
	return ui.Dialog{
		Open: page.directory.Get,
		OnOpenChange: func(open bool, _ base.Reason) {
			page.directory.Set(open)
		},
		Label:  title,
		TestID: "provider-directory-modal",
		Bare:   true,
		Inset:  16,
		Style: core.Style{
			Width:        core.Pt(464),
			MaxWidth:     core.Pct(100),
			MaxHeight:    core.Pt(488),
			CornerRadius: 14,
			BorderCurve:  core.BorderCurveContinuous,
			BorderWidth:  1,
		},
		StyleFunc: func() core.Style {
			shadow := []core.BoxShadow{{
				Offset: core.Point{Y: 14},
				Blur:   32,
				Color:  core.RGBA(0x161918, 0.18),
			}, {
				Offset: core.Point{Y: 3},
				Blur:   10,
				Color:  core.RGBA(0x161918, 0.08),
			}}
			if cx.Dark() {
				shadow = []core.BoxShadow{{
					Offset: core.Point{Y: 14},
					Blur:   32,
					Color:  core.RGBA(0x000000, 0.60),
				}}
			}
			return core.Style{
				Background:  colors.Background(),
				BorderColor: colors.Border(),
				Shadows:     shadow,
			}
		},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				var searchField core.Ref
				return box("flex-1 min-h-0 bg-background",
					box("flex-row items-center gap-3 px-5 pt-4 pb-3.5 bg-surface border-b "+divider,
						box("size-9 rounded-xl bg-accent-tint items-center justify-center shrink-0", glyph(headerIcon, accent)),
						box("flex-1 min-w-0 gap-0.5",
							label(title, "text-base font-semibold "+ink),
							label(subtitle, "text-xs "+muted),
						),
						iconButton(cx, "关闭弹窗", lucide.X(), closeDialog),
					),
					box("flex-1 min-h-0 px-4 pt-3.5 pb-3 gap-3",
						searchBar(cx, &searchField, "搜索服务", "搜索服务名称", page.search),
						core.Scroll{
							Label: "可选服务列表",
							Class: "flex-1 min-h-0",
							Style: core.Style{MaxHeight: core.Pt(276)},
							Children: []core.Node{
								core.Dynamic(func(cx *core.Scope) core.Node {
									query := strings.ToLower(strings.TrimSpace(page.search.Get()))
									var builtIn, configured []core.Node
									for i := range service.Providers {
										provider := &service.Providers[i]
										if provider.AI != isAI || !strings.Contains(strings.ToLower(provider.Name), query) && !strings.Contains(provider.ID, query) {
											continue
										}
										name := strings.TrimSuffix(provider.Name, "（内置）")
										var badges []core.Node
										if provider.BuiltIn {
											badges = append(badges, pill("免密钥", "bg-accent-tint text-accent"))
										}
										if a.hasProvider(provider.ID) {
											badges = append(badges, pill("已添加", "bg-inset text-text-muted"))
										}
										row := providerRow(cx, name, provider.Name, isAI, badges, func() { pick(provider) })
										if provider.BuiltIn {
											builtIn = append(builtIn, row)
										} else {
											configured = append(configured, row)
										}
									}
									if len(builtIn) == 0 && len(configured) == 0 {
										return box("items-center justify-center gap-1.5 py-8 px-4 "+cardSubtle,
											glyph(lucide.Search(), faint),
											label("没有匹配的服务", "text-sm font-medium "+muted),
										)
									}
									if isAI {
										return box("gap-2 pr-1 pb-1", configured...)
									}
									var groups []core.Node
									for _, group := range []struct {
										title string
										rows  []core.Node
									}{{"免配置", builtIn}, {"需配置", configured}} {
										if len(group.rows) > 0 {
											groups = append(groups, box("gap-2",
												label(group.title, "pl-1 text-xs font-semibold "+muted),
												box("gap-2", group.rows...),
											))
										}
									}
									return box("gap-3.5 pr-1 pb-1", groups...)
								}),
							},
						},
					),
					box("flex-row items-center justify-between gap-2 px-5 py-3 bg-surface border-t "+divider,
						label("选择服务后将进入详细配置", "text-xs "+muted),
						compactAction(cx, "取消", nil, closeDialog),
					),
				)
			}),
		},
	}
}

func providerRow(cx *core.Scope, title, a11yLabel string, isAI bool, badges []core.Node, open func()) base.Button {
	icon := lucide.Globe()
	if isAI {
		icon = lucide.Cpu()
	}
	content := []core.Node{label(title, "text-sm font-medium "+ink)}
	if len(badges) > 0 {
		content = append(content, box("flex-row flex-wrap items-center gap-1.5", badges...))
	}
	b := base.Button{
		Label:   a11yLabel,
		OnPress: func(core.PressEvent) { open() },
		Class:   "min-h-12 px-3.5 py-2.5 flex-row items-center gap-3 " + sheet + " " + buttonStates,
		Children: []core.Node{
			box("size-8 rounded-lg bg-accent-tint items-center justify-center shrink-0", glyphSm(icon, accent)),
			box("flex-1 min-w-0 gap-1", content...),
			glyphSm(lucide.ChevronRight(), faint),
		},
	}
	b.StyleFunc = func() core.Style { return rowCardMotion(cx) }
	return b
}

func (a *application) builtInProviderForm(cx *core.Scope, provider *service.Provider, status *core.Signal[string], back func()) core.Node {
	pref, _ := a.providerSettings(provider.ID)
	button := "启用翻译"
	if pref.Translate {
		button = "停用翻译"
	}
	return box("gap-3 pb-4",
		box("flex-row", compactAction(cx, "返回服务列表", lucide.ChevronLeft(), back)),
		card(cx, "p-4 gap-3.5",
			box("flex-row items-center gap-3",
				box("size-9 rounded-lg bg-accent-tint items-center justify-center", glyph(lucide.Globe(), accent)),
				box("gap-0.5 flex-1 min-w-0",
					label(provider.Name, "text-base font-semibold "+ink),
					label("免配置在线翻译来源", "text-xs "+muted),
				),
			),
			label("无需账号或密钥。启用后，原文会发送给此在线翻译服务。", "text-sm "+muted),
			box("flex-row pt-1", action(cx, button, nil, !pref.Translate, func() {
				if err := a.saveProvider(cx.Secrets(), provider.ID, providerPreference{Translate: !pref.Translate}, nil, false); err != nil {
					status.Set(err.Error())
					return
				}
				back()
				status.Set("服务设置已保存")
			})),
		),
	)
}

func (a *application) providerForm(cx *core.Scope, provider *service.Provider, draft *providerDraft, chat, status *core.Signal[string], back func()) core.Node {
	translate := core.NewSignal(cx, draft.pref.Translate)
	save := func(makeChat bool) {
		err := a.saveProvider(cx.Secrets(), provider.ID, draft.pref, draft.secrets, makeChat)
		chat.Set(a.chatProviderID())
		if err != nil {
			status.Set(err.Error())
			return
		}
		status.Set("服务设置已保存")
	}
	headerIcon := lucide.Globe()
	subtitle := "云翻译服务配置"
	if provider.AI {
		headerIcon = lucide.Cpu()
		subtitle = "AI 模型服务配置"
	}
	fields := []core.Node{
		box("flex-row items-center gap-3 pb-2 border-b "+subtleBorder,
			box("size-9 rounded-lg bg-accent-tint items-center justify-center", glyph(headerIcon, accent)),
			box("gap-0.5 flex-1 min-w-0",
				label(provider.Name, "text-base font-semibold "+ink),
				label(subtitle, "text-xs "+muted),
			),
		),
	}
	for _, field := range provider.Fields {
		values := draft.pref.Fields
		if field.Secret {
			values = draft.secrets
		}
		fields = append(fields, box("gap-1.5",
			label(field.Label, "text-xs font-semibold "+ink),
			ui.Input{
				Style:        core.Style{Height: core.Pt(36), FontSize: 13},
				Label:        field.Label,
				DefaultValue: values[field.Key],
				Masked:       field.Secret,
				OnChange:     func(value string) { values[field.Key] = value },
			},
		))
	}
	fields = append(fields, box("flex-row items-center justify-between gap-2 pt-3 border-t "+subtleBorder,
		box("gap-0.5",
			label("参与在线翻译", "text-sm font-medium "+ink),
			label("开启后将作为多源翻译之一同时返回译文。", "text-xs "+muted),
		),
		ui.Switch{
			Label:   "参与在线翻译",
			Size:    ui.SizeSmall,
			Checked: translate.Get,
			OnChange: func(enabled bool) {
				draft.pref.Translate = enabled
				translate.Set(enabled)
			},
		},
	))
	return box("gap-3 pb-4",
		box("flex-row", compactAction(cx, "返回服务列表", lucide.ChevronLeft(), back)),
		card(cx, "p-4 gap-3.5", fields...),
		box("flex-row flex-wrap items-center gap-2",
			action(cx, "保存服务设置", lucide.Check(), true, func() { save(false) }),
			core.Show{When: func() bool { return provider.AI && chat.Get() != provider.ID }, Then: func(cx *core.Scope) core.Node {
				return action(cx, "保存并用于问答", lucide.MessageCircle(), false, func() { save(true) })
			}},
			core.Show{When: func() bool { return provider.AI && chat.Get() == provider.ID }, Then: func(cx *core.Scope) core.Node {
				return popReveal(cx, 0, pill("当前问答提供商", "border border-accent-border bg-accent-tint text-accent"))
			}},
		),
	)
}
