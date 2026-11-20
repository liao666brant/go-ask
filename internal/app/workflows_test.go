package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/nsgotest"
	"goask/internal/data"
	"goask/internal/service"
)

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
	panel := nsgotest.Start(t, a.root, nsgotest.Size(660, 510))
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
		return nsgotest.Start(t, a.root, nsgotest.Size(660, 510))
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
