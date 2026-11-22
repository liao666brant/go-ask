package app

import (
	"context"
	"strings"

	"goask/internal/data"

	"github.com/nsgo-dev/nsgo/core"
)

type application struct {
	host                       *desktopHost
	db                         *data.Store
	configPath                 string
	pref                       preferences
	keys                       credentials
	credentialsLoaded          bool
	hotkeyError, keyStoreError string

	cx               *core.Scope
	lifetime         context.Context
	editor           core.Ref
	page, mode, note *core.Signal[string]
	pinned           *core.Signal[bool]

	source, target, query *core.Signal[string]
	resultsVersion        *core.Signal[int]
	entry                 data.Entry
	found                 bool
	submitted             translationQuery
	translations          []translation
	translationID         int
	translateCtx          context.Context
	cancelTranslation     context.CancelFunc

	question, partial *core.Signal[string]
	generating        *core.Signal[bool]
	messagesVersion   *core.Signal[int]
	conversation      int64
	messages          []data.Message
	chatErrors        map[int64]string
	generationID      int
	cancelChat        context.CancelFunc
}

func (a *application) desktopHotkey() string { return a.pref.Hotkey }

func (a *application) onHotkeyError(err error) {
	if err == nil {
		return
	}
	a.hotkeyError = "无法注册 " + strings.ToUpper(a.pref.Hotkey) + "：" + err.Error()
	if a.note != nil {
		a.note.Set(a.hotkeyError)
	}
}
func (a *application) focusEditor() {
	if content := a.host.content; content != nil {
		content.Post(func() { a.editor.Focus() })
	}
}
func (a *application) onDesktopShow(selection string) {
	a.mode.Set("translate")
	a.page.Set("work")
	if strings.TrimSpace(selection) != "" {
		a.query.Set(selection)
		a.translate()
	} else {
		a.clearTranslation()
		a.query.Set("")
		a.note.Set(a.hotkeyError)
	}
	a.focusEditor()
}

func (a *application) initUI(cx *core.Scope) {
	a.cx = cx
	a.lifetime = cx.Context()
	a.page = core.NewSignal(cx, "work")
	a.mode = core.NewSignal(cx, "translate")
	a.source = core.NewSignal(cx, "auto")
	a.target = core.NewSignal(cx, "smart")
	a.query = core.NewSignal(cx, "")
	a.question = core.NewSignal(cx, "")
	a.note = core.NewSignal(cx, a.hotkeyError)
	a.pinned = core.NewSignal(cx, false)
	a.partial = core.NewSignal(cx, "")
	a.resultsVersion = core.NewSignal(cx, 0)
	a.messagesVersion = core.NewSignal(cx, 0)
	a.chatErrors = make(map[int64]string)
	a.generating = core.NewSignal(cx, false)
	hist, err := a.db.Conversations("")
	if err != nil {
		a.note.Set(err.Error())
	} else if len(hist) > 0 {
		a.openConversation(hist[0].ID)
	}
}

func (a *application) switchMode(mode string) {
	a.page.Set("work")
	a.note.Set("")
	a.mode.Set(mode)
	a.focusEditor()
}

func increment(v int) int { return v + 1 }
