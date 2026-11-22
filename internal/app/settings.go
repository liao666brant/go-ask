package app

import (
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

type settingsPage int

const (
	settingsGeneral settingsPage = iota
	settingsTranslation
	settingsAI
)

func (a *application) settingsView(cx *core.Scope) core.Node {
	cx.SetTheme(workbenchTheme)
	a.loadCredentials(cx)
	window := desktop.CurrentWindow(cx)
	cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways}, window.Close)
	page := core.NewSignal(cx, settingsGeneral)
	generalStatus := core.NewSignal(cx, a.hotkeyError)
	serviceStatus := core.NewSignal(cx, a.keyStoreError)
	general := a.generalSettings(cx, generalStatus)
	providers := a.providerSettingsView(cx, serviceStatus)

	nav := func(title string, icon *core.SVG, target settingsPage) core.Node {
		return base.Button{
			Label:    title,
			Selected: func() bool { return page.Get() == target },
			Class:    "flex-row items-center gap-2 px-3 h-11 rounded-md text-text-muted " + buttonStates + " " + selectedFill + " aria-selected:text-accent",
			OnPress:  func(core.PressEvent) { page.Set(target) },
			Children: []core.Node{glyph(icon, ""), label(title, "text-sm font-semibold")},
		}
	}
	sidebar := box("shrink-0 p-3 gap-1 border-r "+divider+" bg-surface",
		box("px-2 pt-3 pb-5 flex-row items-center gap-2", brandMark(), label("Go Ask", "text-lg font-semibold "+ink)),
		nav("常规", lucide.Settings2(), settingsGeneral),
		nav("翻译服务", lucide.Languages(), settingsTranslation),
		nav("AI 提供商", lucide.MessageCircle(), settingsAI),
		box("flex-1"),
		action("退出 Go Ask", nil, false, a.host.quit),
	)
	sidebar.Style.Width = core.Pt(152)
	return box("flex-1 flex-row bg-background", sidebar,
		core.Dynamic(func(*core.Scope) core.Node {
			selected := page.Get()
			if selected == settingsGeneral {
				return settingsBody("常规", generalStatus, general)
			}
			if selected == settingsAI {
				return settingsBody("AI 提供商", serviceStatus, providers(true))
			}
			return settingsBody("翻译服务", serviceStatus, providers(false))
		}),
	)
}

func settingsBody(title string, status *core.Signal[string], content core.Node) core.Node {
	return box("flex-1 min-w-0",
		box("px-5 pt-5 pb-4 border-b "+divider, label(title, "text-2xl font-semibold "+ink)),
		core.Show{When: func() bool { return status.Get() != "" }, Then: func(*core.Scope) core.Node {
			return box("mx-4 mt-3 px-3 py-2 "+sheet,
				core.Text{Role: core.RoleStatus, Class: "text-sm " + ink, ContentFunc: status.Get})
		}},
		core.Scroll{Label: title + "内容", Class: "flex-1", Children: []core.Node{box("px-4 pt-4", content)}},
	)
}

func (a *application) generalSettings(cx *core.Scope, status *core.Signal[string]) core.Node {
	key := core.NewSignal(cx, a.pref.Hotkey)
	panelIdle := core.NewSignal(cx, a.pref.PanelIdleSeconds)
	hideOnBlur := core.NewSignal(cx, a.pref.HidePanelOnBlur)
	recording := core.NewSignal(cx, false)
	var recorder core.Ref
	hotkey := core.View{Ref: &recorder, Role: core.RoleButton, Label: "录入唤起快捷键", Class: "h-10 px-3 flex-row gap-2 items-center justify-center rounded-md border bg-inset " + divider + " " + buttonStates, OnPress: func(core.PressEvent) { recording.Set(true); recorder.Focus() }, OnBlur: func() { recording.Set(false) }, OnKeyDown: func(e core.KeyEvent) bool {
		if !recording.Get() {
			return false
		}
		if e.Key == "escape" {
			recording.Set(false)
			return true
		}
		if combo := e.Shortcut(); combo != "" {
			if !strings.Contains(combo, "+") || combo == "ctrl+tab" || combo == "ctrl+n" {
				status.Set("请选择带修饰键的组合，避开窗口内快捷键。")
				return true
			}
			key.Set(combo)
			recording.Set(false)
			status.Set("")
		}
		return true
	}, Children: []core.Node{glyph(lucide.Keyboard(), accent), core.Text{Class: "font-mono text-sm font-medium " + ink, ContentFunc: func() string {
		if recording.Get() {
			return "请按新的组合键…"
		}
		return strings.ToUpper(key.Get())
	}}}}
	applyKey := func() {
		value := key.Get()
		a.host.changeHotkey(value, func(err error) {
			if err != nil {
				status.Set(err.Error())
				return
			}
			cfg := a.pref
			cfg.Hotkey = value
			if err := a.savePreferences(cfg); err != nil {
				status.Set("快捷键未能保存：" + err.Error())
				a.host.changeHotkey(a.pref.Hotkey, func(restore error) {
					if restore != nil {
						a.onHotkeyError(restore)
					}
				})
				return
			}
			if a.hotkeyError != "" && a.note != nil && a.note.GetUntracked() == a.hotkeyError {
				a.note.Set("")
			}
			a.hotkeyError = ""
			status.Set("快捷键已生效")
		})
	}
	const settingRow = "px-3 py-3 min-h-12 flex-row items-center justify-between gap-2 flex-wrap"
	apply := action("应用快捷键", lucide.Check(), true, applyKey)
	return box("gap-4 pb-4",
		box("gap-2",
			label("快捷键", "pl-1 text-sm font-semibold "+ink),
			box("p-3 gap-3 "+sheet,
				label("唤起 / 划词快捷键", "text-sm font-semibold "+ink),
				label("点击组合键开始录入，按 Esc 取消录入。", "text-sm "+muted),
				box("flex-row flex-wrap items-center gap-2", hotkey, apply)),
		),
		box("gap-2", label("窗口行为", "pl-1 text-sm font-semibold "+ink), box(sheet,
			box(settingRow, box("flex-1 min-w-[144] gap-1",
				label("失焦时收起主窗口", "text-sm font-medium "+ink),
				label("置顶期间保持显示。", "text-xs "+muted)), ui.Switch{
				Label: "失焦时收起主窗口", Checked: hideOnBlur.Get,
				Size: ui.SizeSmall,
				OnChange: func(enabled bool) {
					cfg := a.pref
					cfg.HidePanelOnBlur = enabled
					if err := a.savePreferences(cfg); err != nil {
						status.Set("面板设置未能保存：" + err.Error())
						return
					}
					hideOnBlur.Set(enabled)
					if enabled && a.host.panel.Visible() && !a.pinned.GetUntracked() {
						a.host.hide()
					}
					status.Set("面板设置已保存")
				},
			}),
			box(settingRow+" border-t "+divider, box("flex-1 min-w-[144] gap-1",
				label("收起后释放面板", "text-sm font-medium "+ink),
				label("释放窗口资源后，仍可用快捷键唤起。", "text-xs "+muted)), ui.Select[int]{
				Label: "收起后释放面板",
				Options: []ui.Option[int]{
					{Value: 10, Label: "10 秒"}, {Value: 30, Label: "30 秒"},
					{Value: 60, Label: "1 分钟"}, {Value: 0, Label: "从不"},
				},
				Style: core.Style{Width: core.Pt(116), Height: core.Pt(36)},
				Value: panelIdle.Get,
				OnChange: func(seconds int) {
					cfg := a.pref
					cfg.PanelIdleSeconds = seconds
					if err := a.savePreferences(cfg); err != nil {
						status.Set("面板设置未能保存：" + err.Error())
						return
					}
					panelIdle.Set(seconds)
					a.host.schedulePanelRelease()
					status.Set("面板设置已保存")
				},
			}))),
	)
}

func aboutView(cx *core.Scope) core.Node {
	cx.SetTheme(workbenchTheme)
	closeWindow := desktop.CurrentWindow(cx).Close
	cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways}, closeWindow)
	version := core.App().Version
	if version == "" {
		version = "开发版"
	} else {
		version = "版本 " + version
	}
	return box("flex-1 justify-between p-4 gap-3 bg-background",
		box("p-4 gap-3 "+sheet,
			box("flex-row gap-3 items-center", brandMark(),
				box("gap-1", label("Go Ask", "text-xl font-semibold "+ink), label(version, "text-sm "+muted))),
			label("桌面划词翻译与 AI 快速问答", "text-base "+ink),
			label("使用 NSGo 构建，离线英汉词条来自 ECDICT。", "text-sm "+muted)),
		box("flex-row justify-end", action("关闭", nil, true, closeWindow)))
}
