package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTranslationServices(t *testing.T) {
	original := "hello + world\nagain"
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		switch r.URL.Path {
		case "/v2/translate":
			if r.Host != "api-free.deepl.com" || r.Header.Get("Authorization") != "DeepL-Auth-Key test:fx" {
				t.Errorf("incorrect DeepL authentication or host")
			}
			var data map[string]any
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				t.Error(err)
			}
			if data["target_lang"] != "ZH-HANS" || data["source_lang"] != nil || data["text"].([]any)[0] != original {
				t.Errorf("wrong DeepL body %v", data)
			}
			fmt.Fprint(w, `{"translations":[{"text":"你好"}]}`)
		case "/api/trans/vip/translate":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			salt := r.Form.Get("salt")
			expect := fmt.Sprintf("%x", md5.Sum([]byte("app"+original+salt+"secret")))
			if r.Host != "fanyi-api.baidu.com" || r.URL.RawQuery != "" || r.Form.Get("from") != "auto" || r.Form.Get("to") != "zh" || r.Form.Get("q") != original || salt == "" || r.Form.Get("sign") != expect {
				t.Error("wrong Baidu form, host or signature")
			}
			fmt.Fprint(w, `{"trans_result":[{"dst":"你好"},{"dst":"世界"}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer mock.Close()
	testURL, _ := url.Parse(mock.URL)
	prior := client
	client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Host = r.URL.Host
		r.URL.Host, r.URL.Scheme = testURL.Host, testURL.Scheme
		return http.DefaultTransport.RoundTrip(r)
	})}
	t.Cleanup(func() { client = prior })
	cfg := Config{DeepLKey: "test:fx", BaiduAppID: "app", BaiduSecret: "secret"}
	for provider, want := range map[string]string{DeepL: "你好", Baidu: "你好\n世界"} {
		result, err := Translate(context.Background(), cfg, provider, original, "auto", "zh")
		if err != nil || result != want {
			t.Errorf("%s = %q, %v", provider, result, err)
		}
	}
}

func TestAIRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("invalid AI URL or authorization")
		}
		var input struct {
			Model    string
			Messages []Message
			Stream   bool
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
			return
		}
		if input.Model != "test" || input.Messages[0].Role != "system" || input.Messages[len(input.Messages)-1].Role != "user" {
			t.Errorf("invalid request: %+v", input)
		}
		if !input.Stream {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"翻译结果"}}]}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		fmt.Fprint(w, ": heartbeat\r\n\r\ndata: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\r\n\r\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"好\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer srv.Close()
	cfg := Config{BaseURL: srv.URL + "/v1/", APIKey: "token", Model: "test"}
	result, err := Translate(context.Background(), cfg, AI, "example", "en", "zh")
	if err != nil || result != "翻译结果" {
		t.Fatalf("translation = %q, %v", result, err)
	}
	var pieces []string
	err = Chat(context.Background(), cfg, []Message{{"user", "问题"}}, func(s string) { pieces = append(pieces, s) })
	if err != nil || strings.Join(pieces, "|") != "你|好" {
		t.Fatalf("chat = %v, %v", pieces, err)
	}
}

func TestInterruptedStream(t *testing.T) {
	raw := "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"
	var piece string
	err := readStream(strings.NewReader(raw), func(s string) { piece += s })
	if !errors.Is(err, io.ErrUnexpectedEOF) || piece != "partial" {
		t.Fatalf("interrupted stream = %q, %v", piece, err)
	}
}

func TestChatCancellation(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Chat(ctx, Config{BaseURL: srv.URL, Model: "test"}, []Message{{"user", "q"}}, func(string) {})
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request never started")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancellation not propagated")
	}
}

func TestAIServiceErrorAndJSONFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/valid/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"choices":[{"message":{"content":"完整回答"}}]}`)
		default:
			http.Error(w, "provider-private-error", http.StatusTooManyRequests)
		}
	}))
	defer srv.Close()
	var result string
	cfg := Config{BaseURL: srv.URL + "/valid/chat/completions", Model: "test"}
	sendChat := func() error {
		return Chat(context.Background(), cfg, []Message{{"user", "问题"}}, func(s string) { result += s })
	}
	if err := sendChat(); err != nil || result != "完整回答" {
		t.Fatalf("JSON stream fallback = %q, %v", result, err)
	}
	cfg.BaseURL = srv.URL
	err := sendChat()
	if err == nil || !strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "provider-private-error") {
		t.Fatalf("HTTP error = %v", err)
	}
}
