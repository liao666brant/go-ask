package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestHidePanelOnBlurPreference(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	firstRun, err := loadPreferences(path)
	if err != nil || !firstRun.HidePanelOnBlur {
		t.Fatalf("first run defaults to hiding on blur: %v, %v", firstRun.HidePanelOnBlur, err)
	}
	if err := os.WriteFile(path, []byte(`{"Hotkey":"ctrl+shift+q"}`), 0600); err != nil {
		t.Fatal(err)
	}
	pref, err := loadPreferences(path)
	if err != nil || !pref.HidePanelOnBlur || pref.Hotkey != "ctrl+shift+q" {
		t.Fatalf("legacy preferences not loaded with blur default: %+v, %v", pref, err)
	}
	a := &application{configPath: dir, pref: pref, host: &desktopHost{}}
	settings := nsgotest.Start(t, a.settingsView, nsgotest.Size(580, 620))
	const control = "label:失焦时收起主窗口"
	settings.Press(control)
	settings.Find("text:面板设置未能保存")
	if !a.pref.HidePanelOnBlur || !settings.Find(control).Props.Checked {
		t.Fatal("save failure changed the active preference")
	}
	a.configPath = path
	settings.Press(control)
	if settings.Find(control).Props.Checked {
		t.Fatal("switch remains on after saving off")
	}
	reloaded, err := loadPreferences(path)
	if err != nil || reloaded.HidePanelOnBlur {
		t.Fatalf("stored off value not restored: %v, %v", reloaded.HidePanelOnBlur, err)
	}
	settings.Post(func() {
		err = a.saveProvider(core.Secrets{}, service.MTran, providerPreference{
			Fields: map[string]string{"baseURL": "http://localhost:8989"},
		}, nil, false)
	})
	settings.Settle()
	// The standalone settings fixture has no host keychain.
	if err == nil || a.pref.HidePanelOnBlur {
		t.Fatal("failed service save modified the blur preference")
	}
	reloaded, err = loadPreferences(path)
	if err != nil || reloaded.HidePanelOnBlur {
		t.Fatalf("saving service settings overwrote the blur preference: %v, %v", reloaded.HidePanelOnBlur, err)
	}
}

func TestServiceSettingsCommitEndpointAndCredentialsTogether(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	initial := preferences{BaseURL: "https://first.test", Model: "legacy-model", Hotkey: "ctrl+q",
		DeepL: true, Baidu: false, AI: false, BaiduAppID: "baidu-app"}
	legacy := application{configPath: path}
	if err := legacy.savePreferences(initial); err != nil {
		t.Fatal(err)
	}
	var cx *core.Scope
	panel := nsgotest.Start(t, func(scope *core.Scope) core.Node {
		cx = scope
		return core.Text{}
	}, nsgotest.Secrets(map[string]string{"providers": `{"APIKey":"first-fixture-key","DeepLKey":"deepl-fixture-key","BaiduSecret":"baidu-fixture-key"}`}))
	reopen := func() *application {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(raw, []byte("fixture-key")) {
			t.Fatal("settings file contains a service key")
		}
		pref, err := loadPreferences(path)
		if err != nil {
			t.Fatal(err)
		}
		a := &application{configPath: path, pref: pref}
		panel.Post(func() { a.loadCredentials(cx) })
		panel.Settle()
		return a
	}
	assertPair := func(a *application, url, key string) {
		t.Helper()
		cfg, err := a.providerConfig(service.AI)
		if err != nil || !a.credentialsLoaded || cfg.BaseURL != url || cfg.APIKey != key {
			t.Fatal("service settings and keychain account are out of sync")
		}
		deepl, deeplKeys := a.providerSettings(service.DeepL)
		baidu, baiduKeys := a.providerSettings(service.Baidu)
		if !deepl.Translate || deeplKeys["apiKey"] != "deepl-fixture-key" ||
			baidu.Translate || baidu.Fields["appID"] != "baidu-app" || baiduKeys["appSecret"] != "baidu-fixture-key" ||
			a.chatProviderID() != service.AI {
			t.Fatal("legacy migration lost a provider or its saved settings")
		}
	}
	a := reopen()
	assertPair(a, initial.BaseURL, "first-fixture-key")
	next, _ := a.providerSettings(service.AI)
	next.Fields["baseURL"] = "https://second.test"
	keys := map[string]string{"apiKey": "second-fixture-key"}
	var err error
	save := func() {
		panel.Post(func() { err = a.saveProvider(cx.Secrets(), service.AI, next, keys, false) })
		panel.Settle()
	}
	panel.Post(func() { err = a.saveProvider(core.Secrets{}, service.AI, next, keys, false) })
	panel.Settle()
	if err == nil {
		t.Fatal("expected keychain write to fail")
	}
	assertPair(a, initial.BaseURL, "first-fixture-key")
	assertPair(reopen(), initial.BaseURL, "first-fixture-key")

	a.configPath = dir // A settings file cannot replace an existing directory.
	save()
	if err == nil {
		t.Fatal("expected settings rename to fail")
	}
	assertPair(a, initial.BaseURL, "first-fixture-key")
	assertPair(reopen(), initial.BaseURL, "first-fixture-key")

	a.configPath = path
	save()
	if err != nil {
		t.Fatal(err)
	}
	assertPair(a, next.Fields["baseURL"], "second-fixture-key")
	assertPair(reopen(), next.Fields["baseURL"], "second-fixture-key")

	// The open settings window still holds its original preferences snapshot.
	updated := a.pref
	updated.PanelIdleSeconds = 10
	updated.HidePanelOnBlur = false
	if err := a.savePreferences(updated); err != nil {
		t.Fatal(err)
	}
	next.Fields["model"] = "updated"
	save()
	if err != nil {
		t.Fatal(err)
	}
	assertPair(reopen(), next.Fields["baseURL"], "second-fixture-key")
	if latest := reopen().pref; latest.PanelIdleSeconds != 10 || latest.HidePanelOnBlur {
		t.Fatal("saving a service settings draft reverted a panel preference")
	}
	delete(keys, "apiKey")
	save()
	if err != nil {
		t.Fatal(err)
	}
	assertPair(reopen(), next.Fields["baseURL"], "")
	panel.Post(func() {
		if _, exists, err := cx.Secrets().Get("providers-alternate"); err != nil || exists {
			t.Error("previous keychain entry was not removed")
		}
	})
	panel.Settle()
}

func TestProviderCredentialsStoredSeparately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	var scope *core.Scope
	window := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		scope = cx
		return core.Text{}
	}, nsgotest.Secrets(map[string]string{}))
	a := &application{configPath: path, pref: preferences{Providers: map[string]providerPreference{}}}
	firstKey, secondKey := strings.Repeat("a", 1400), strings.Repeat("b", 1400)
	var saveErr error
	window.Post(func() {
		a.loadCredentials(scope)
		for _, entry := range []struct{ id, key string }{{service.AI, firstKey}, {service.DeepSeek, secondKey}} {
			draft, _ := a.providerSettings(entry.id)
			draft.Fields["model"] = "fixture"
			if err := a.saveProvider(scope.Secrets(), entry.id, draft, map[string]string{"apiKey": entry.key}, false); err != nil {
				saveErr = err
				return
			}
		}
	})
	window.Settle()
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	pref, err := loadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := &application{pref: pref}
	window.Post(func() { reopened.loadCredentials(scope) })
	window.Settle()
	for _, entry := range []struct{ id, key string }{{service.AI, firstKey}, {service.DeepSeek, secondKey}} {
		cfg, err := reopened.providerConfig(entry.id)
		if err != nil || cfg.APIKey != entry.key || reopened.pref.Providers[entry.id].CredentialSlot != 1 {
			t.Fatal("the provider's credential did not survive independently")
		}
	}
}

func TestLocalProviderNeedsNoCredentialStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	pref, err := loadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	a := &application{configPath: path, pref: pref}
	draft, _ := a.providerSettings(service.MTran)
	draft.Translate = true
	if err := a.saveProvider(core.Secrets{}, service.MTran, draft, nil, false); err != nil {
		t.Fatal(err)
	}
	pref, err = loadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := pref.Providers[service.MTran]
	if !entry.Translate || entry.CredentialSlot != 0 || entry.Fields["baseURL"] != "http://localhost:8989" {
		t.Fatalf("unexpected local provider settings: %+v", entry)
	}
}

func TestMigratingLocalProviderDoesNotCreateEmptyCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	const localURL = "http://localhost:1234/v1"
	a := &application{configPath: path, pref: preferences{
		BaseURL: localURL, Model: "local-model", AI: true,
	}}
	var scope *core.Scope
	window := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		scope = cx
		return core.Text{}
	})
	var saveErr error
	window.Post(func() {
		a.loadCredentials(scope)
		draft, _ := a.providerSettings(service.MTran)
		saveErr = a.saveProvider(scope.Secrets(), service.MTran, draft, nil, false)
	})
	window.Settle()
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	pref, err := loadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := &application{pref: pref}
	cfg, err := reopened.providerConfig(service.AI)
	entry := pref.Providers[service.AI]
	if err != nil || cfg.BaseURL != localURL ||
		!entry.Translate || entry.CredentialSlot != 0 || reopened.chatProviderID() != service.AI {
		t.Fatalf("legacy local provider no longer works without a credential store: %v", err)
	}
}
