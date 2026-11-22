package app

import (
	"path/filepath"
	"testing"

	"goask/internal/data"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestSystemAppearanceUpdatesWindowsInPlace(t *testing.T) {
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	a := &application{db: db}
	a.host = &desktopHost{app: a}
	env := core.Environment{OS: core.OSWindows, Idiom: core.IdiomDesktop}
	panel := nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight), nsgotest.Environment(env))
	panel.Press("label:问答")
	panel.Type("role:textbox", "待发送的问题")
	editor := panel.Find("role:textbox")
	active := panel.Find("label:问答")

	_, settings := openSettingsFixture(t)
	settings.Press("label:翻译服务")
	section := settings.Find("label:翻译服务")
	addProvider := settings.Find("label:添加服务")
	if active.Style.Background != workbenchTheme.Light.Selection ||
		section.Style.Background != workbenchTheme.Light.Selection ||
		addProvider.Style.Background != workbenchTheme.Light.Accent {
		t.Fatal("light palette did not style the mode, settings section, and primary action")
	}

	env.Dark = true
	panel.SetEnvironment(env)
	settings.SetEnvironment(env)
	currentEditor := panel.Find("role:textbox")
	currentSection := settings.Find("label:翻译服务")
	if currentEditor.ID != editor.ID || currentEditor.Text != "待发送的问题" ||
		panel.Find("label:问答").Style.Background != workbenchTheme.Dark.Selection {
		t.Fatal("dark appearance did not preserve the composer and update the selected mode")
	}
	if currentSection.ID != section.ID || !currentSection.Props.Selected ||
		currentSection.Style.Background != workbenchTheme.Dark.Selection ||
		settings.Find("label:添加服务").Style.Background != workbenchTheme.Dark.Accent {
		t.Fatal("dark appearance did not preserve and restyle the active settings section")
	}
}
