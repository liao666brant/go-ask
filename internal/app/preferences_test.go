package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestServiceSettingsCommitEndpointAndCredentialsTogether(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	initial := preferences{BaseURL: "https://first.test", Hotkey: "ctrl+q"}
	legacy := application{configPath: path}
	if err := legacy.savePreferences(initial); err != nil {
		t.Fatal(err)
	}
	var cx *core.Scope
	panel := nsgotest.Start(t, func(scope *core.Scope) core.Node {
		cx = scope
		return core.Text{}
	}, nsgotest.Secrets(map[string]string{"providers": `{"APIKey":"first-fixture-key"}`}))
	reopen := func() *application {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(raw, []byte("fixture-key")) {
			t.Fatal("settings file contains a service key")
		}
		a := &application{configPath: path}
		if err := json.Unmarshal(raw, &a.pref); err != nil {
			t.Fatal(err)
		}
		panel.Post(func() { a.loadCredentials(cx) })
		panel.Settle()
		return a
	}
	assertPair := func(a *application, url, key string) {
		t.Helper()
		cfg := a.serviceConfig()
		if !a.credentialsLoaded || cfg.BaseURL != url || cfg.APIKey != key {
			t.Fatal("service settings and keychain account are out of sync")
		}
	}
	a := reopen()
	assertPair(a, initial.BaseURL, "first-fixture-key")
	next := a.pref
	next.BaseURL = "https://second.test"
	keys := credentials{APIKey: "second-fixture-key"}
	var err error
	save := func() {
		panel.Post(func() { err = a.saveServiceSettings(cx.Secrets(), next, keys) })
		panel.Settle()
	}
	panel.Post(func() { err = a.saveServiceSettings(core.Secrets{}, next, keys) })
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
	assertPair(a, next.BaseURL, "second-fixture-key")
	assertPair(reopen(), next.BaseURL, "second-fixture-key")

	// The open settings window still holds its original preferences snapshot.
	next.Model = "updated"
	save()
	if err != nil {
		t.Fatal(err)
	}
	assertPair(reopen(), next.BaseURL, "second-fixture-key")
	keys.APIKey = ""
	save()
	if err != nil {
		t.Fatal(err)
	}
	assertPair(reopen(), next.BaseURL, "")
	panel.Post(func() {
		if _, exists, err := cx.Secrets().Get("providers-alternate"); err != nil || exists {
			t.Error("previous keychain entry was not removed")
		}
	})
	panel.Settle()
}
