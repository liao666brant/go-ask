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

func (a *application) providerSettingsView(cx *core.Scope, status *core.Signal[string]) func(bool) core.Node {
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

	return func(isAI bool) core.Node {
		page := translation
		if isAI {
			page = ai
		}
		return core.Dynamic(func(cx *core.Scope) core.Node {
			pick := func(provider *service.Provider) {
				page.directory.Set(false)
				page.selected.Set(provider)
				status.Set("")
			}
			if provider := page.selected.Get(); provider != nil {
				if provider.BuiltIn {
					return a.builtInProviderForm(cx, provider, status, func() { pick(nil) })
				}
				draft := drafts[provider.ID]
				if draft == nil {
					pref, secrets := a.providerSettings(provider.ID)
					draft = &providerDraft{pref: pref, secrets: secrets}
					drafts[provider.ID] = draft
				}
				return a.providerForm(cx, provider, draft, chat, status, func() { pick(nil) })
			}
			if page.directory.Get() {
				return providerDirectory(page, isAI, pick)
			}

			add := action("添加服务", lucide.Plus(), true, func() {
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
				state := "翻译已停用"
				if pref.Translate {
					state = "已启用翻译"
				}
				if isAI && chat.Get() == provider.ID {
					state += " · 当前问答提供商"
				}
				items = append(items, providerRow(provider.Name, state, func() { pick(provider) }))
			}
			if len(items) == 0 {
				hint := "添加免配置来源，即可开始在线翻译。"
				if isAI {
					hint = "添加提供商并选择用于问答。"
				}
				items = append(items, box("p-4 gap-1 "+sheet,
					label("尚未添加服务", "text-base font-semibold "+ink),
					label(hint, "text-sm "+muted)))
			}
			return box("pb-4 gap-3", box("flex-row justify-end", add), box("gap-2", items...))
		})
	}
}

func providerDirectory(page providerPage, isAI bool, pick func(*service.Provider)) core.Node {
	var searchField core.Ref
	return box("gap-3 pb-4",
		box("flex-row", action("返回服务列表", lucide.ChevronLeft(), false, func() { pick(nil) })),
		box("flex-row items-center gap-2",
			glyph(lucide.Search(), muted),
			ui.Input{
				Ref: &searchField, Label: "搜索服务", Placeholder: "搜索服务名称", AutoFocus: true,
				Class: "flex-1", Value: page.search.Get, OnChange: page.search.Set,
				Style: core.Style{Height: core.Pt(40)},
			},
			core.Show{When: func() bool { return page.search.Get() != "" }, Then: func(*core.Scope) core.Node {
				return iconButton("清除搜索", lucide.X(), func() {
					page.search.Set("")
					searchField.Focus()
				})
			}},
		),
		core.Dynamic(func(*core.Scope) core.Node {
			query := strings.ToLower(strings.TrimSpace(page.search.Get()))
			var builtIn, configured []core.Node
			for i := range service.Providers {
				provider := &service.Providers[i]
				if provider.AI != isAI || !strings.Contains(strings.ToLower(provider.Name), query) && !strings.Contains(provider.ID, query) {
					continue
				}
				row := providerRow(strings.TrimSuffix(provider.Name, "（内置）"), "", func() { pick(provider) })
				row.Label = provider.Name
				if provider.BuiltIn {
					builtIn = append(builtIn, row)
				} else {
					configured = append(configured, row)
				}
			}
			if len(builtIn) == 0 && len(configured) == 0 {
				return box("p-4 "+sheet, label("没有匹配的服务", "text-sm "+muted))
			}
			if isAI {
				return box("gap-2", configured...)
			}
			var groups []core.Node
			for _, group := range []struct {
				title string
				rows  []core.Node
			}{{"免配置", builtIn}, {"需配置", configured}} {
				if len(group.rows) > 0 {
					groups = append(groups, box("gap-2",
						label(group.title, "pl-1 text-sm font-semibold "+muted),
						box("gap-2", group.rows...),
					))
				}
			}
			return box("gap-4", groups...)
		}),
	)
}

func providerRow(name, state string, open func()) base.Button {
	content := []core.Node{label(name, "text-sm font-medium "+ink)}
	if state != "" {
		content = append(content, label(state, "text-xs "+muted))
	}
	return base.Button{
		Label: name, OnPress: func(core.PressEvent) { open() },
		Class:    "min-h-12 px-4 py-3 flex-row items-center gap-3 " + sheet + " " + buttonStates,
		Children: []core.Node{box("flex-1 min-w-0 gap-1", content...), glyph(lucide.ChevronRight(), muted)},
	}
}

func (a *application) builtInProviderForm(cx *core.Scope, provider *service.Provider, status *core.Signal[string], back func()) core.Node {
	pref, _ := a.providerSettings(provider.ID)
	button := "启用翻译"
	if pref.Translate {
		button = "停用翻译"
	}
	return box("gap-3 pb-4",
		box("flex-row", action("返回服务列表", lucide.ChevronLeft(), false, back)),
		box("p-4 gap-3 "+sheet,
			label(provider.Name, "text-lg font-semibold "+ink),
			label("无需账号或密钥。启用后，原文会发送给此在线翻译服务。", "text-sm "+muted),
			box("flex-row", action(button, nil, !pref.Translate, func() {
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
	fields := []core.Node{label(provider.Name, "text-lg font-semibold "+ink)}
	for _, field := range provider.Fields {
		values := draft.pref.Fields
		if field.Secret {
			values = draft.secrets
		}
		fields = append(fields, box("gap-1",
			label(field.Label, "text-sm font-medium "+ink),
			ui.Input{
				Style: core.Style{Height: core.Pt(36)},
				Label: field.Label, DefaultValue: values[field.Key], Masked: field.Secret,
				OnChange: func(value string) { values[field.Key] = value },
			},
		))
	}
	fields = append(fields, box("flex-row items-center justify-between gap-2",
		label("参与在线翻译", "text-sm "+ink),
		ui.Switch{Label: "参与在线翻译", Size: ui.SizeSmall, Checked: translate.Get, OnChange: func(enabled bool) {
			draft.pref.Translate = enabled
			translate.Set(enabled)
		}},
	))
	return box("gap-3 pb-4",
		box("flex-row", action("返回服务列表", lucide.ChevronLeft(), false, back)),
		box("p-4 gap-3 "+sheet, fields...),
		box("flex-row flex-wrap items-center gap-2",
			action("保存服务设置", lucide.Check(), true, func() { save(false) }),
			core.Show{When: func() bool { return provider.AI && chat.Get() != provider.ID }, Then: func(*core.Scope) core.Node {
				return action("保存并用于问答", nil, false, func() { save(true) })
			}},
			core.Show{When: func() bool { return provider.AI && chat.Get() == provider.ID }, Then: func(*core.Scope) core.Node {
				return label("当前问答提供商", "text-sm "+muted)
			}},
		),
	)
}
