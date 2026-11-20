package app

import (
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"goask/internal/service"
)

func (a *application) settingsView(cx *core.Scope) core.Node {
	a.loadCredentials(cx)
	window := desktop.CurrentWindow(cx)
	cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways}, window.Close)
	next, keys := a.pref, a.keys
	status := core.NewSignal(cx, a.hotkeyError)
	if status.GetUntracked() == "" {
		status.Set(a.keyStoreError)
	}
	key := core.NewSignal(cx, next.Hotkey)
	recording := core.NewSignal(cx, false)
	var recorder core.Ref
	cx.Post(recorder.Focus)
	startRecord := func() { recording.Set(true); recorder.Focus() }
	hotkey := core.View{Ref: &recorder, Role: core.RoleButton, Label: "录入唤起快捷键", Class: "h-9 px-3 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-950", OnPress: func(core.PressEvent) { startRecord() }, OnBlur: func() { recording.Set(false) }, OnKeyDown: func(e core.KeyEvent) bool {
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
	}, Children: []core.Node{core.Text{Class: "font-semibold " + accent, ContentFunc: func() string {
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
	field := func(title string, dst *string, masked bool, placeholder string) core.Node {
		return box("gap-1", label(title, "text-xs font-semibold "+muted),
			core.Input{Class: "w-full h-9 px-2 rounded-md bg-slate-100 dark:bg-slate-700 " + ink, Label: title, DefaultValue: *dst, Masked: masked, Placeholder: placeholder, OnChange: func(v string) { *dst = v }})
	}
	toggle := func(title string, dst *bool) core.Node {
		value := core.NewSignal(cx, *dst)
		return core.View{Role: core.RoleButton, Label: title, Class: "flex-row items-center gap-2 self-start", OnPress: func(core.PressEvent) { *dst = !*dst; value.Set(*dst) }, Children: []core.Node{core.Text{Class: "text-base font-semibold " + accent, ContentFunc: func() string {
			if value.Get() {
				return "☑  " + title
			}
			return "☐  " + title
		}}}}
	}
	save := func() {
		next.BaseURL = strings.TrimSpace(next.BaseURL)
		next.Model = strings.TrimSpace(next.Model)
		next.BaiduAppID = strings.TrimSpace(next.BaiduAppID)
		if next.BaseURL != "" {
			if _, err := service.ChatEndpoint(next.BaseURL); err != nil {
				status.Set(err.Error())
				return
			}
		}
		if err := a.saveServiceSettings(cx.Secrets(), next, keys); err != nil {
			status.Set(err.Error())
			return
		}
		status.Set("服务设置已保存")
	}
	return box("flex-1 px-4 py-3 gap-2 bg-slate-50 dark:bg-slate-900", label("设置", "text-xl font-semibold "+ink), core.Text{Class: "text-xs " + muted, ContentFunc: status.Get},
		core.Scroll{Class: "flex-1", Children: []core.Node{box("gap-3 pb-3",
			box("gap-2 p-3 "+card, label("唤起 / 划词快捷键", "font-semibold "+ink), box("flex-row items-center gap-2", hotkey, action("应用快捷键", lucide.Check(), false, applyKey))),
			box("p-3 gap-2 "+card, toggle("DeepL 翻译", &next.DeepL), field("DeepL API Key", &keys.DeepLKey, true, "支持 Free / Pro")),
			box("p-3 gap-2 "+card, toggle("百度翻译", &next.Baidu), field("百度 App ID", &next.BaiduAppID, false, ""), field("百度密钥", &keys.BaiduSecret, true, "")),
			box("p-3 gap-2 "+card, toggle("AI 翻译", &next.AI), field("AI 服务地址（翻译和问答共用）", &next.BaseURL, false, "https://api.openai.com/v1"), field("模型名称", &next.Model, false, "使用服务端提供的模型 ID"), field("AI API Key", &keys.APIKey, true, "本地无鉴权服务可留空")),
		)}}, box("flex-row items-center", action("退出 Go Ask", nil, false, a.host.quit), box("flex-1"), action("保存服务设置", lucide.Check(), true, save)))
}

func aboutView(cx *core.Scope) core.Node {
	closeWindow := desktop.CurrentWindow(cx).Close
	cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways}, closeWindow)
	version := core.App().Version
	if version == "" {
		version = "开发版"
	} else {
		version = "版本 " + version
	}
	return box("flex-1 p-5 gap-3 bg-slate-50 dark:bg-slate-900",
		box("flex-row gap-3 items-center", box("size-10 rounded-lg items-center justify-center bg-indigo-600", glyph(lucide.Languages(), "text-white")),
			box("gap-1", label("Go Ask", "text-xl font-semibold "+ink), label(version, "text-sm "+muted))),
		label("桌面划词翻译与 AI 快速问答", "text-base "+ink),
		label("使用 NSGo 构建，离线英汉词条来自 ECDICT。", "text-sm "+muted),
		box("flex-1"), box("flex-row justify-end", action("关闭", nil, false, closeWindow)))
}
