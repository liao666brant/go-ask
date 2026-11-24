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
			Class:    "flex-row items-center gap-2.5 px-3 h-9 rounded-lg text-text-muted " + buttonStates + " " + selectedFill + " aria-selected:text-accent",
			Style:    buttonMotion(cx, 0.975),
			OnPress:  func(core.PressEvent) { page.Set(target) },
			Children: []core.Node{glyphSm(icon, ""), label(title, "text-sm font-medium")},
		}
	}
	quitButton := base.Button{
		Label:   "退出 Go Ask",
		Class:   "flex-row items-center justify-center gap-2 h-9 px-3 rounded-lg border bg-surface-subtle text-text-muted " + subtleBorder + " " + buttonStates,
		Style:   buttonMotion(cx, 0.975),
		OnPress: func(core.PressEvent) { a.host.quit() },
		Children: []core.Node{
			glyphSm(lucide.LogOut(), muted),
			label("退出 Go Ask", "text-xs font-medium "+muted),
		},
	}
	sidebar := box("shrink-0 p-3 gap-1 border-r "+divider+" bg-surface",
		box("px-2 pt-2 pb-4 flex-row items-center gap-2.5",
			brandMark(),
			box("gap-0.5",
				label("Go Ask", "text-sm font-semibold "+ink),
				label("偏好设置", "text-xs "+muted),
			),
		),
		nav("常规", lucide.Settings2(), settingsGeneral),
		nav("翻译服务", lucide.Languages(), settingsTranslation),
		nav("AI 提供商", lucide.MessageCircle(), settingsAI),
		box("flex-1"),
		box("pt-2 border-t "+subtleBorder, quitButton),
	)
	sidebar.Style.Width = core.Pt(152)
	return box("flex-1 flex-row bg-background", sidebar,
		core.Dynamic(func(cx *core.Scope) core.Node {
			selected := page.Get()
			if selected == settingsGeneral {
				return enterRevealFlex(cx, 6, settingsBody(cx, "常规", "配置全局唤起快捷键与浮窗驻留行为", generalStatus, general))
			}
			if selected == settingsAI {
				return enterRevealFlex(cx, 6, settingsBody(cx, "AI 提供商", "管理用于问答与 AI 翻译的模型服务", serviceStatus, providers(true)))
			}
			return enterRevealFlex(cx, 6, settingsBody(cx, "翻译服务", "管理离线词典之外的在线翻译来源", serviceStatus, providers(false)))
		}),
	)
}

func settingsBody(cx *core.Scope, title, subtitle string, status *core.Signal[string], content core.Node) core.Node {
	return box("flex-1 min-w-0",
		box("px-5 pt-4 pb-3.5 bg-surface border-b gap-0.5 "+divider,
			label(title, "text-xl font-semibold "+ink),
			label(subtitle, "text-xs "+muted),
		),
		core.Show{When: func() bool { return status.Get() != "" }, Then: func(cx *core.Scope) core.Node {
			return popReveal(cx, -4, box("mx-4 mt-3 flex-row items-center gap-2 px-3.5 py-2.5 rounded-lg border border-accent-border bg-accent-tint",
				glyphSm(lucide.Info(), accent),
				core.Text{Role: core.RoleStatus, Class: "flex-1 min-w-0 text-sm " + ink, ContentFunc: status.Get},
			))
		}},
		core.Scroll{Label: title + "内容", Class: "flex-1", Children: []core.Node{box("px-4 pt-4", content)}},
	)
}

func (a *application) generalSettings(cx *core.Scope, status *core.Signal[string]) core.Node {
	key := core.NewSignal(cx, a.pref.Hotkey)
	panelIdle := core.NewSignal(cx, a.pref.PanelIdleSeconds)
	hideOnBlur := core.NewSignal(cx, a.pref.HidePanelOnBlur)
	recording := core.NewSignal(cx, false)
	colors := ui.ColorsOf(cx)
	var recorder core.Ref
	hotkey := core.View{
		Ref:   &recorder,
		Role:  core.RoleButton,
		Label: "录入唤起快捷键",
		Class: "h-9 px-3.5 flex-row gap-2 items-center justify-center rounded-lg border " + buttonStates,
		Style: buttonMotion(cx, 0.98),
		StyleFunc: func() core.Style {
			if recording.Get() {
				return core.Style{
					Background:  cx.Palette().Selection,
					BorderColor: colors.Accent(),
				}
			}
			return core.Style{
				Background:  colors.Inset(),
				BorderColor: colors.Border(),
			}
		},
		OnPress: func(core.PressEvent) { recording.Set(true); recorder.Focus() },
		OnBlur:  func() { recording.Set(false) },
		OnKeyDown: func(e core.KeyEvent) bool {
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
		},
		Children: []core.Node{
			glyphSm(lucide.Keyboard(), accent),
			core.Text{Class: "font-mono text-sm font-semibold " + ink, ContentFunc: func() string {
				if recording.Get() {
					return "请按新的组合键…"
				}
				return strings.ToUpper(key.Get())
			}},
		},
	}
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
	const settingRow = "px-3.5 py-3 min-h-12 flex-row items-center justify-between gap-2 flex-wrap"
	apply := action(cx, "应用快捷键", lucide.Check(), true, applyKey)
	return box("gap-4 pb-4",
		box("gap-2",
			label("快捷键", "pl-1 text-xs font-semibold "+muted),
			card(cx, "p-3.5 gap-3",
				box("gap-0.5",
					label("唤起 / 划词快捷键", "text-sm font-semibold "+ink),
					label("点击右侧按键区域开始录入，按 Esc 取消。", "text-xs "+muted),
				),
				box("flex-row flex-wrap items-center gap-2", hotkey, apply),
			),
		),
		box("gap-2",
			label("窗口行为", "pl-1 text-xs font-semibold "+muted),
			card(cx, "overflow-hidden",
				box(settingRow,
					core.View{
						Class: "flex-1 gap-0.5",
						Style: core.Style{MinWidth: core.Pt(144)},
						Children: []core.Node{
							label("失焦时收起主窗口", "text-sm font-medium "+ink),
							label("置顶期间保持显示。", "text-xs "+muted),
						},
					},
					ui.Switch{
						Label:   "失焦时收起主窗口",
						Checked: hideOnBlur.Get,
						Size:    ui.SizeSmall,
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
					},
				),
				box(settingRow+" border-t "+subtleBorder,
					core.View{
						Class: "flex-1 gap-0.5",
						Style: core.Style{MinWidth: core.Pt(144)},
						Children: []core.Node{
							label("收起后释放面板", "text-sm font-medium "+ink),
							label("释放窗口资源后，仍可用快捷键唤起。", "text-xs "+muted),
						},
					},
					ui.Select[int]{
						Label: "收起后释放面板",
						Options: []ui.Option[int]{
							{Value: 10, Label: "10 秒"}, {Value: 30, Label: "30 秒"},
							{Value: 60, Label: "1 分钟"}, {Value: 0, Label: "从不"},
						},
						Style: core.Style{Width: core.Pt(116), Height: core.Pt(34)},
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
					},
				),
			),
		),
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
		version = "v" + version
	}
	return box("flex-1 justify-between p-5 gap-4 bg-background",
		card(cx, "p-5 gap-4",
			box("flex-row gap-3.5 items-center",
				box("size-12 rounded-2xl items-center justify-center bg-accent", glyph(lucide.Terminal(), "text-accent-text")),
				box("gap-1",
					box("flex-row items-center gap-2",
						label("Go Ask", "text-xl font-semibold "+ink),
						pill(version, "bg-accent-tint text-accent"),
					),
					label("桌面划词翻译与 AI 快速问答", "text-sm "+muted),
				),
			),
			box("pt-3 border-t gap-1 "+subtleBorder,
				label("原生桌面引擎由 NSGo 驱动，离线英汉词条来自 ECDICT。", "text-xs "+muted),
			),
		),
		box("flex-row justify-end", action(cx, "关闭", nil, true, closeWindow)),
	)
}
