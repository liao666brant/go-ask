package app

import (
	"path/filepath"
	"testing"
	"time"

	"goask/internal/data"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestInteractionMotionRespectsReducedMotion(t *testing.T) {
	openPanel := func(t *testing.T, reduced bool) *nsgotest.App {
		t.Helper()
		db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		a := &application{db: db}
		a.host = &desktopHost{app: a}
		return nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight), nsgotest.Environment(core.Environment{
			OS:                core.OSWindows,
			Idiom:             core.IdiomDesktop,
			CanSetWindowLevel: true,
			ReduceMotion:      reduced,
		}))
	}

	t.Run("standard motion", func(t *testing.T) {
		panel := openPanel(t, false)
		btn := panel.Find("id:translation-submit")
		if btn.Style.Transition.Duration <= 0 || btn.Style.Transition.Properties&core.TransitionTransform == 0 {
			t.Fatalf("primary action missing transform transition: %+v", btn.Style.Transition)
		}

		// Trigger dictionary lookup and verify animated copy feedback enters and exits within 2s.
		panel.Type("role:textbox", "hello")
		panel.Submit("role:textbox")
		panel.Find("text:离线英汉词典")
		panel.Press("label:复制释义")
		panel.Find("text:已复制")
		panel.Advance(2 * time.Second)
		if panel.Exists("text:已复制") {
			t.Fatal("copy toast did not finish exit animation within 2s")
		}
	})

	t.Run("reduced motion", func(t *testing.T) {
		panel := openPanel(t, true)
		btn := panel.Find("id:translation-submit")
		if btn.Style.Transition.Duration <= 0 || btn.Style.Transition.Properties&core.TransitionTransform != 0 {
			t.Fatalf("reduced motion must keep color-only transitions without transform: %+v", btn.Style.Transition)
		}
		panel.Type("role:textbox", "hello")
		panel.Submit("role:textbox")
		panel.Find("text:离线英汉词典")
		panel.Press("label:复制释义")
		panel.Find("text:已复制")
		panel.Advance(2 * time.Second)
		if panel.Exists("text:已复制") {
			t.Fatal("reduced motion copy toast did not clear at 2s")
		}
	})
}
