package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"goask/internal/data"
	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestChatWithoutProviderKeepsDraft(t *testing.T) {
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{db: db, pref: preferences{Providers: map[string]providerPreference{}}}
	a.host = &desktopHost{app: a}
	panel := nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight))
	if panel.Enabled("id:translation-submit") {
		t.Fatal("translation should require a non-blank draft")
	}
	panel.Key("ctrl+tab")
	if panel.Enabled("label:发送") {
		t.Fatal("send should require a non-blank draft")
	}
	panel.Type("role:textbox", "待发送的问题")
	panel.Press("label:发送")
	panel.Find("text:请先在设置中配置并选择问答提供商")
	panel.Find("text:待发送的问题")
	if a.conversation != 0 {
		t.Fatal("sending without a provider created a conversation")
	}
}

func TestChatScrollLetsUsersReadEarlierMessages(t *testing.T) {
	db, err := data.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{db: db}
	a.host = &desktopHost{app: a}
	panel := nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight))
	answer := strings.Repeat("这是用于滚动验证的一行旧回答。\n", 30)
	panel.Post(func() {
		a.messages = []data.Message{{Role: "assistant", Content: answer}}
		a.generating.Set(true)
		a.messagesVersion.Update(increment)
	})
	panel.Key("ctrl+tab")
	const scroll = "label:会话消息"
	const jump = "label:查看最新消息"
	if panel.Find(scroll).Offset.Y == 0 {
		t.Fatal("opening chat should reveal the latest messages")
	}
	panel.ScrollTo(scroll, core.Point{})
	panel.Find(jump)
	panel.Post(func() { a.partial.Set(strings.Repeat("流式回答\n", 10)) })
	panel.Settle()
	if panel.Find(scroll).Offset.Y != 0 {
		t.Fatal("a streamed reply moved the scroll position while reading earlier messages")
	}
	panel.Press(jump)
	if panel.Find(scroll).Offset.Y == 0 || panel.Exists(jump) {
		t.Fatal("jump should reveal the latest message and resume following")
	}
	panel.ScrollTo(scroll, core.Point{})
	panel.Press("label:复制回答")
	panel.Find("text:已复制")
	if panel.Clipboard != answer {
		t.Fatal("copy did not preserve the full answer")
	}
	panel.Advance(2 * time.Second)
	if panel.Exists("text:已复制") {
		t.Fatal("copy feedback did not clear")
	}
}

func TestChatProviderSwitchKeepsInFlightAnswerAndContext(t *testing.T) {
	finish := make(chan struct{})
	release := sync.OnceFunc(func() { close(finish) })
	second := make(chan []service.Message, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Model    string
			Stream   bool
			Messages []service.Message
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || !request.Stream {
			t.Error("expected a streaming chat request")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/first/chat/completions":
			if request.Model != "first-model" || r.Header.Get("Authorization") != "Bearer first-key" {
				t.Error("first request used another provider's configuration")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"private reasoning\"}}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"第一段\"}}]}\n\n")
			w.(http.Flusher).Flush()
			select {
			case <-finish:
			case <-r.Context().Done():
				return
			}
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"结束\"}}]}\n\ndata: [DONE]\n\n")
		case "/second/chat/completions":
			if request.Model != "second-model" || r.Header.Get("Authorization") != "Bearer second-key" {
				t.Error("follow-up used another provider's configuration")
			}
			second <- request.Messages
			fmt.Fprint(w, `{"choices":[{"message":{"content":"第二家回答"}}]}`)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer release()
	d := t.TempDir()
	db, err := data.Open(filepath.Join(d, "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{
		db: db, configPath: filepath.Join(d, "settings.json"), credentialsLoaded: true,
		pref: preferences{ChatProvider: service.DeepSeek, Providers: map[string]providerPreference{
			service.DeepSeek: {Fields: map[string]string{"baseURL": server.URL + "/first", "model": "first-model"}},
			service.Zhipu:    {Fields: map[string]string{"baseURL": server.URL + "/second", "model": "second-model"}},
		}},
		keys: credentials{Providers: map[string]map[string]string{
			service.DeepSeek: {"apiKey": "first-key"}, service.Zhipu: {"apiKey": "second-key"},
		}},
	}
	a.host = &desktopHost{app: a}
	panel := nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight))
	panel.Key("ctrl+tab")
	panel.Type("role:textbox", "第一个问题")
	panel.Press("label:发送")
	panel.WaitFor("text:第一段", 3*time.Second)
	panel.Post(func() {
		draft, keys := a.providerSettings(service.Zhipu)
		err = a.saveProvider(core.Secrets{}, service.Zhipu, draft, keys, true)
	})
	panel.Settle()
	if err != nil {
		t.Fatal(err)
	}
	release()
	panel.WaitFor("text:第一段结束", 3*time.Second)
	panel.WaitFor("label:发送", 3*time.Second)
	if panel.Exists("text:private reasoning") {
		t.Fatal("reasoning text was shown as the answer")
	}
	panel.Type("role:textbox", "追问")
	panel.Press("label:发送")
	panel.WaitFor("text:第二家回答", 3*time.Second)
	got := <-second
	want := []service.Message{
		{Role: "user", Content: "第一个问题"}, {Role: "assistant", Content: "第一段结束"},
		{Role: "user", Content: "追问"},
	}
	if len(got) != len(want)+1 || !slices.Equal(got[1:], want) {
		t.Fatalf("follow-up context: %+v", got)
	}
}

func TestBackgroundChatErrorStaysWithConversation(t *testing.T) {
	finish := make(chan struct{})
	release := sync.OnceFunc(func() { close(finish) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"部分回答"}}]}`+"\n\n")
		w.(http.Flusher).Flush()
		<-finish // Closing the stream without a completion marker is an error.
	}))
	defer server.Close()
	defer release()
	db, err := data.Open(filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &application{db: db, pref: preferences{BaseURL: server.URL, Model: "fixture"}}
	a.host = &desktopHost{app: a}
	panel := nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight))
	panel.Key("ctrl+tab")
	panel.Type("text:有什么想问的？", "查询命令")
	panel.Press("label:发送")
	panel.WaitFor("text:部分回答", 3*time.Second)
	panel.Key("ctrl+tab")
	release()
	panel.WaitUntil(3*time.Second, func() bool { return !a.generating.GetUntracked() })
	if panel.Exists("text:问答失败") {
		t.Fatal("chat failure appeared in translation mode")
	}
	panel.Key("ctrl+tab")
	panel.Find("text:问答失败")
	panel.Key("ctrl+n")
	if panel.Exists("text:问答失败") {
		t.Fatal("a new conversation inherited the previous failure")
	}
	panel.Press("label:历史会话")
	panel.Press("label:查询命令")
	panel.Find("text:问答失败")
}

func TestChatWindowRemembersConversation(t *testing.T) {
	const first = "Which process uses port 8080?"
	const follow = "How do I see its PID?"
	const timeout = 4 * time.Second
	finish := make(chan struct{})
	release := sync.OnceFunc(func() { close(finish) })
	turn := make(chan []service.Message, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []service.Message
			Stream   bool
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !req.Stream || len(req.Messages) == 0 {
			http.Error(w, "invalid chat request", http.StatusBadRequest)
			return
		}
		turn <- req.Messages
		w.Header().Set("Content-Type", "text/event-stream")
		delta := func(s string) {
			_, _ = fmt.Fprintf(w, `data: {"choices":[{"delta":{"content":%q}}]}`+"\n\n", s)
			w.(http.Flusher).Flush()
		}
		switch req.Messages[len(req.Messages)-1].Content {
		case first:
			delta("Use ss")
			select {
			case <-finish:
			case <-r.Context().Done():
				return
			}
			delta(" for sockets.")
		case follow:
			delta("Add -p for the PID.")
		default:
			t.Errorf("unexpected prompt: %+v", req.Messages)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	defer release()
	path := filepath.Join(t.TempDir(), "chat.db")
	openWindow := func(t *testing.T) *nsgotest.App {
		t.Helper()
		db, err := data.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Error(err)
			}
		})
		a := &application{db: db, pref: preferences{BaseURL: server.URL + "/v1", Model: "fixture"}}
		a.host = &desktopHost{app: a}
		return nsgotest.Start(t, a.root, nsgotest.Size(panelWidth, panelMaxHeight))
	}
	if !t.Run("send and follow up", func(t *testing.T) {
		app := openWindow(t)
		app.Type("text:输入单词或粘贴一段文字…", "hello")
		app.Submit("text:hello")
		app.Find("text:离线英汉词典")
		app.Key("ctrl+tab")
		app.Type("text:有什么想问的？", first)
		app.Submit("text:" + first)
		app.WaitFor(`text:"Use ss"`, timeout)
		app.Find("label:停止")
		release()
		app.WaitFor("text:Use ss for sockets.", timeout)
		app.WaitFor("label:发送", timeout)
		<-turn
		app.Type("text:有什么想问的？", follow)
		app.Press("label:发送")
		app.WaitFor("text:Add -p for the PID.", timeout)
		app.WaitFor("label:发送", timeout)
		msgs := <-turn
		var actual []service.Message
		for _, m := range msgs {
			if m.Role != "system" {
				actual = append(actual, m)
			}
		}
		want := []service.Message{{Role: "user", Content: first}, {Role: "assistant", Content: "Use ss for sockets."}, {Role: "user", Content: follow}}
		if !slices.Equal(actual, want) {
			t.Fatalf("follow-up context: %+v", actual)
		}
		app.Key("ctrl+tab")
		app.Find("text:离线英汉词典")
	}) {
		return
	}
	t.Run("reopen and search saved answers", func(t *testing.T) {
		app := openWindow(t)
		app.Press("label:历史会话")
		app.Type("text:搜索标题或对话内容", "the PID")
		app.Press("label:" + first)
		app.Find("text:Use ss for sockets.")
		app.Find("text:" + follow)
		app.Find("text:Add -p for the PID.")
	})
}
