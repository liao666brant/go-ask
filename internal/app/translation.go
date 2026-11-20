package app

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"goask/internal/data"
	"goask/internal/service"
)

type translation struct {
	provider, value, failure string
	pending                  bool
}

// The submitted query stays with its results while the user edits the next draft.
type translationQuery struct {
	text, source, target string
}

func (a *application) clearTranslation() {
	if a.cancelTranslation != nil {
		a.cancelTranslation()
	}
	a.translationID++
	a.found = false
	a.translations = nil
	a.submitted = translationQuery{}
	a.resultsVersion.Update(increment)
}
func targetLanguage(source, target, text string) string {
	if target != "smart" {
		return target
	}
	if source == "zh" {
		return "en"
	}
	if source == "auto" {
		han := false
		for _, r := range text {
			if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
				return "zh"
			}
			han = han || unicode.In(r, unicode.Han)
		}
		if han {
			return "en"
		}
	}
	return "zh"
}
func (a *application) translate() {
	text := strings.TrimSpace(a.query.Get())
	if text == "" {
		return
	}
	source := a.source.Get()
	query := translationQuery{text: text, source: source, target: targetLanguage(source, a.target.Get(), text)}
	a.startTranslation(query, false)
}

func (a *application) startTranslation(query translationQuery, forceOnline bool) {
	a.clearTranslation()
	a.submitted = query
	a.note.Set("")
	if (query.source == "auto" || query.source == "en") && query.target == "zh" {
		a.entry, a.found = data.Lookup(query.text)
	}
	a.resultsVersion.Update(increment)
	if a.found && !forceOnline {
		return
	}
	a.loadCredentials(a.cx)
	cfg := a.serviceConfig()
	for _, p := range []struct {
		id      string
		enabled bool
	}{
		{service.DeepL, a.pref.DeepL && cfg.DeepLKey != ""},
		{service.Baidu, a.pref.Baidu && cfg.BaiduAppID != "" && cfg.BaiduSecret != ""},
		{service.AI, a.pref.AI && cfg.Model != "" && cfg.BaseURL != ""},
	} {
		if p.enabled {
			a.translations = append(a.translations, translation{provider: p.id, pending: true})
		}
	}
	if len(a.translations) == 0 {
		a.note.Set("请在设置中启用并配置至少一个翻译服务。")
		return
	}
	ctx, cancel := context.WithCancel(a.lifetime)
	a.cancelTranslation = cancel
	a.translateCtx = ctx
	a.resultsVersion.Update(increment)
	for i := range a.translations {
		a.requestTranslation(ctx, i)
	}
}
func (a *application) requestTranslation(ctx context.Context, i int) {
	run := a.translationID
	cfg := a.serviceConfig()
	p := a.translations[i].provider
	query := a.submitted
	go func() {
		value, err := service.Translate(ctx, cfg, p, query.text, query.source, query.target)
		a.cx.Post(func() {
			if run != a.translationID {
				return
			}
			r := &a.translations[i]
			r.pending = false
			r.value = value
			if err != nil {
				r.failure = err.Error()
			} else {
				r.failure = ""
			}
			a.resultsVersion.Update(increment)
		})
	}()
}
func (a *application) askAbout(text string) {
	a.newConversation()
	a.question.Set(fmt.Sprintf("原文：%s\n译文：%s\n\n", a.submitted.text, text))
}
