package app

import (
	"path/filepath"
	"testing"

	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func openSettingsFixture(t *testing.T) (*application, *nsgotest.App) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	pref, err := loadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	a := &application{pref: pref, configPath: path}
	settings := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		a.host = &desktopHost{app: a, scope: cx}
		return a.settingsView(cx)
	}, nsgotest.Size(580, 620), nsgotest.Environment(core.Environment{OS: core.OSWindows, Idiom: core.IdiomDesktop}))
	return a, settings
}

func TestSettingsNavigationPreservesGeneralActions(t *testing.T) {
	a, settings := openSettingsFixture(t)
	if !settings.Find("label:常规").Props.Selected {
		t.Fatal("general page must be selected when opening settings")
	}
	settings.Focus("label:翻译服务")
	settings.Key("space")
	if !settings.Find("label:翻译服务").Props.Selected || settings.Find("label:常规").Props.Selected {
		t.Fatal("keyboard navigation did not select the translation page")
	}
	settings.Find("label:添加服务")
	if settings.Exists("label:录入唤起快捷键") {
		t.Fatal("general settings remained visible on the translation page")
	}
	settings.Focus("label:AI 提供商")
	settings.Key("enter")
	if !settings.Find("label:AI 提供商").Props.Selected {
		t.Fatal("keyboard navigation did not select the AI page")
	}
	settings.Press("label:常规")
	settings.Resize(400, 340)
	for _, selector := range []string{"label:应用快捷键", "label:失焦时收起主窗口", "label:收起后释放面板"} {
		bounds := settings.Find(selector).Bounds()
		if bounds.Origin.X+bounds.Size.Width > 400 {
			t.Fatalf("%s is clipped at the settings minimum width: %v", selector, bounds)
		}
	}
	settings.Press("label:录入唤起快捷键")
	settings.Key("ctrl+shift+h")
	settings.Press("label:应用快捷键")
	settings.Find("text:快捷键已生效")
	settings.Press("label:失焦时收起主窗口")
	settings.Press("label:收起后释放面板")
	settings.Press("text:1 分钟")
	pref, err := loadPreferences(a.configPath)
	if err != nil || pref.Hotkey != "ctrl+shift+h" || pref.HidePanelOnBlur || pref.PanelIdleSeconds != 60 {
		t.Fatalf("general settings were not saved: %+v, %v", pref, err)
	}
}

func TestSettingsProviderDirectoryAndDrafts(t *testing.T) {
	a, settings := openSettingsFixture(t)
	settings.Press("label:失焦时收起主窗口")
	settings.Press("label:翻译服务")
	settings.Press("label:添加服务")
	settings.Type("label:搜索服务", "does-not-match")
	settings.Find("text:没有匹配的服务")
	settings.Press("label:清除搜索")
	settings.Find("label:MTranServer")
	settings.Type("label:搜索服务", "MTRAN")
	settings.Find("label:MTranServer")
	if settings.Exists("label:腾讯混元翻译") || settings.Exists("label:DeepSeek") {
		t.Fatal("directory includes providers outside the current search or category")
	}
	settings.Press("label:MTranServer")
	settings.Clear("label:服务地址")
	settings.Type("label:服务地址", "not-a-url")
	settings.Press("label:保存服务设置")
	if a.hasProvider(service.MTran) || settings.Exists("text:服务设置已保存") {
		t.Fatal("invalid service address was saved")
	}
	settings.Clear("label:服务地址")
	settings.Type("label:服务地址", "http://127.0.0.1:18989")
	settings.Press("label:参与在线翻译")
	settings.Press("label:返回服务列表")
	settings.Press("label:添加服务")
	settings.Type("label:搜索服务", "mtran")
	settings.Press("label:MTranServer")
	settings.Find("text:http://127.0.0.1:18989")

	settings.Press("label:AI 提供商")
	settings.Press("label:添加服务")
	settings.Type("label:搜索服务", "deepseek")
	settings.Press("label:DeepSeek")
	settings.Press("label:保存服务设置")
	if a.hasProvider(service.DeepSeek) || settings.Exists("text:服务设置已保存") {
		t.Fatal("provider without required configuration was saved")
	}
	settings.Type("label:模型 ID", "fixture-model")
	settings.Type("label:API Key", "fixture-key")
	settings.Press("label:保存并用于问答")
	settings.Find("text:当前问答提供商")
	if a.chatProviderID() != service.DeepSeek || a.hasProvider(service.MTran) {
		t.Fatal("saving the AI provider applied changes from another provider's draft")
	}

	settings.Press("label:翻译服务")
	settings.Find("text:http://127.0.0.1:18989")
	if !settings.Find("label:参与在线翻译").Props.Checked {
		t.Fatal("the translation draft lost its enabled state")
	}
	settings.Press("label:保存服务设置")
	settings.Find("text:服务设置已保存")
	settings.Press("label:返回服务列表")
	settings.Find("text:已启用翻译")
	settings.Press("label:添加服务")
	settings.Type("label:搜索服务", "mtran")
	settings.Press("label:MTranServer")
	settings.Find("text:http://127.0.0.1:18989")
	settings.Press("label:返回服务列表")
	settings.Find("label:MTranServer")
	if a.chatProviderID() != service.DeepSeek || a.pref.HidePanelOnBlur {
		t.Fatal("saving another provider overwrote the AI or general settings")
	}
}

func TestSettingsOpenAIKeyRequiredOnlyForOfficialEndpoint(t *testing.T) {
	a, settings := openSettingsFixture(t)
	settings.Press("label:AI 提供商")
	settings.Press("label:添加服务")
	settings.Press("label:OpenAI 兼容")
	settings.Type("label:模型 ID", "fixture-model")
	settings.Press("label:保存并用于问答")
	settings.Find("text:请填写 OpenAI API Key")
	if a.hasProvider(service.AI) || a.chatProviderID() != "" {
		t.Fatal("official endpoint was saved without a key")
	}
	settings.Clear("label:服务地址")
	settings.Type("label:服务地址", "http://localhost:1234/v1")
	settings.Press("label:保存并用于问答")
	settings.Find("text:当前问答提供商")
	if _, err := a.providerConfig(service.AI); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsBuiltInProviderCanBeEnabledWithoutCredentials(t *testing.T) {
	a, settings := openSettingsFixture(t)
	settings.Press("label:翻译服务")
	settings.Press("label:添加服务")
	settings.Type("label:搜索服务", "谷歌")
	settings.Find("text:免配置")
	settings.Find("label:谷歌 Google Cloud Translation")
	settings.Press("label:谷歌翻译（内置）")
	if a.hasProvider(service.GoogleBuiltIn) || settings.Exists("label:API Key") {
		t.Fatal("selecting a built-in provider should open its credentialless form without enabling it")
	}
	settings.Press("label:启用翻译")
	settings.Find("text:已启用翻译")
	pref, err := loadPreferences(a.configPath)
	entry := pref.Providers[service.GoogleBuiltIn]
	if err != nil || !entry.Translate || entry.CredentialSlot != 0 || len(entry.Fields) != 0 {
		t.Fatalf("built-in preference did not persist without credentials: %+v, %v", entry, err)
	}

	settings.Press("label:谷歌翻译（内置）")
	settings.Press("label:停用翻译")
	settings.Find("text:翻译已停用")
	pref, err = loadPreferences(a.configPath)
	if err != nil || pref.Providers[service.GoogleBuiltIn].Translate {
		t.Fatalf("disabling the built-in provider did not persist: %v", err)
	}
}
