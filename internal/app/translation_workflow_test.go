package app

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/nsgotest"
	"goask/internal/data"
)

type translateTransport func(*http.Request) (*http.Response, error)

func (fn translateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestMultiSourceResponsesAndRetry(t *testing.T) {
	ready := make(chan struct{})
	release := sync.OnceFunc(func() { close(ready) })
	defer release()
	var baidu, deepl atomic.Int32
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	http.DefaultTransport = translateTransport(func(r *http.Request) (*http.Response, error) {
		status := http.StatusOK
		var content string
		switch r.URL.Hostname() {
		case "api.deepl.com":
			deepl.Add(1)
			select {
			case <-ready:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
			content = `{"translations":[{"text":"DeepL 译文"}]}`
		case "fanyi-api.baidu.com":
			if baidu.Add(1) == 1 {
				status = http.StatusTooManyRequests
			} else {
				content = `{"trans_result":[{"dst":"百度译文"}]}`
			}
		case "model.test":
			content = `{"choices":[{"message":{"content":"AI 译文"}}]}`
		default:
			t.Errorf("unexpected provider %s", r.URL.Hostname())
			status = http.StatusNotFound
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(content))}, nil
	})
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{db: db, credentialsLoaded: true, keys: credentials{DeepLKey: "fixture", BaiduSecret: "fixture"}, pref: preferences{DeepL: true, Baidu: true, AI: true, BaiduAppID: "fixture", Model: "fixture", BaseURL: "https://model.test/v1"}}
	a.host = &desktopHost{app: a}
	panel := nsgotest.Start(t, a.root, nsgotest.Size(660, 510))
	panel.Type("text:输入单词或粘贴一段文字…", "a short sentence")
	panel.Submit("text:a short sentence")
	panel.WaitFor("text:AI 译文", 3*time.Second)
	panel.WaitFor("label:重试", 3*time.Second)
	panel.Find("text:翻译中…")
	panel.Press("label:重试")
	panel.WaitFor("text:百度译文", 3*time.Second)
	release()
	panel.WaitFor("text:DeepL 译文", 3*time.Second)
	if baidu.Load() != 2 || deepl.Load() != 1 {
		t.Fatalf("calls after retry: Baidu=%d, DeepL=%d", baidu.Load(), deepl.Load())
	}
}

func TestDictionaryComparisonUsesDisplayedWord(t *testing.T) {
	type payload struct {
		Text   []string
		Source string `json:"source_lang"`
		Target string `json:"target_lang"`
	}
	sent := make(chan payload, 1)
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	http.DefaultTransport = translateTransport(func(r *http.Request) (*http.Response, error) {
		var req payload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
		sent <- req
		body := io.NopCloser(strings.NewReader(`{"translations":[{"text":"online result"}]}`))
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body}, nil
	})
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a := &application{db: db, credentialsLoaded: true, pref: preferences{DeepL: true}, keys: credentials{DeepLKey: "fixture"}}
	a.host = &desktopHost{app: a}
	panel := nsgotest.Start(t, a.root, nsgotest.Size(660, 510))
	panel.Type("text:输入单词或粘贴一段文字…", "hello")
	panel.Submit("text:hello")
	panel.Find("label:在线对照")
	panel.Clear("role:textbox")
	panel.Type("role:textbox", "unsent draft")
	panel.Post(func() { a.source.Set("fr"); a.target.Set("de") })
	panel.Settle()
	panel.Press("label:在线对照")
	panel.WaitFor("text:online result", 3*time.Second)
	panel.Find("text:unsent draft")
	request := <-sent
	if len(request.Text) != 1 || request.Text[0] != "hello" || request.Source != "" || request.Target != "ZH-HANS" {
		t.Fatalf("unexpected comparison request: %+v", request)
	}
}
