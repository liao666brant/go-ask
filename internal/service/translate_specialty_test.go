package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestHunyuanTranslation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Host != "tokenhub.tencentmaas.com" || r.URL.Path != "/v1/api/translations" {
			t.Errorf("unexpected Hunyuan request %s %s%s", r.Method, r.Host, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tencent-key" {
			t.Error("wrong Hunyuan authorization")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		want := map[string]string{"model": "hy-mt2-plus", "text": body["text"], "target": "zh"}
		switch body["text"] {
		case "automatic":
			fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"译文"}}]}`)
		case "explicit":
			want["source"] = "zh-TR"
			fmt.Fprint(w, `{"choices":[{"finish_reason":"sensitive","message":{"content":"blocked"}}]}`)
		default:
			t.Errorf("unexpected text %q", body["text"])
		}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("Hunyuan body = %v, want %v", body, want)
		}
	}))
	t.Cleanup(srv.Close)
	mock, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	prior := client
	client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Host = r.URL.Host
		r.URL.Host, r.URL.Scheme = mock.Host, mock.Scheme
		return http.DefaultTransport.RoundTrip(r)
	})}
	t.Cleanup(func() { client = prior })

	cfg := Config{APIKey: "tencent-key", Model: "hy-mt2-plus"}
	result, err := translateHunyuan(t.Context(), cfg, "automatic", "auto", "zh-Hans")
	if err != nil || result != "译文" {
		t.Fatalf("Hunyuan translation = %q, %v", result, err)
	}
	result, err = translateHunyuan(t.Context(), cfg, "explicit", "zh-TW", "zh")
	if result != "" || err == nil || !strings.Contains(err.Error(), "内容审核") {
		t.Fatalf("Hunyuan sensitive response = %q, %v", result, err)
	}
}

func TestMTranTranslation(t *testing.T) {
	for _, tc := range []struct {
		name, source, from, to, token string
		detectStatus                  int
		wantError                     string
	}{
		{name: "explicit", source: "zh", from: "zh-Hans", to: "de"},
		{name: "detected", source: "auto", from: "fr", to: "zh-Hans", token: "local-token", detectStatus: 200},
		{name: "missing detector", source: "auto", to: "zh-Hans", detectStatus: 404, wantError: "请手动选择源语言"},
		{name: "model unavailable", source: "zh", from: "zh-Hans", to: "fr", wantError: "503"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const original = "原文\n<literal>"
			requests := make(chan string, 4)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.URL.Path
				wantAuth := ""
				if tc.token != "" {
					wantAuth = "Bearer " + tc.token
				}
				if r.Method != http.MethodPost || r.Header.Get("Authorization") != wantAuth {
					t.Errorf("MTran request = %s, unexpected authorization", r.Method)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				switch r.URL.Path {
				case "/proxy/detect":
					if !reflect.DeepEqual(body, map[string]any{"text": original}) {
						t.Errorf("wrong detection body %v", body)
					}
					w.WriteHeader(tc.detectStatus)
					fmt.Fprint(w, `{"language":"fr"}`)
				case "/proxy/translate":
					want := map[string]any{"from": tc.from, "to": tc.to, "text": original, "html": false}
					if !reflect.DeepEqual(body, want) {
						t.Errorf("MTran body = %v, want %v", body, want)
					}
					if tc.name == "model unavailable" {
						http.Error(w, "private deployment details", http.StatusServiceUnavailable)
						return
					}
					fmt.Fprint(w, `{"result":"ligne 1\nligne 2"}`)
				default:
					t.Errorf("unexpected MTran endpoint %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			target := tc.to
			if target == "zh-Hans" {
				target = "zh"
			}
			result, err := translateMTran(t.Context(), Config{APIKey: tc.token, BaseURL: srv.URL + "/proxy/"}, original, tc.source, target)
			if tc.wantError == "" {
				if err != nil || result != "ligne 1\nligne 2" {
					t.Fatalf("MTran translation = %q, %v", result, err)
				}
			} else if result != "" || err == nil || !strings.Contains(err.Error(), tc.wantError) ||
				strings.Contains(err.Error(), "private deployment details") {
				t.Fatalf("MTran error = %q, %v", result, err)
			}
			var want []string
			if tc.detectStatus != 0 {
				want = append(want, "/proxy/detect")
			}
			if tc.detectStatus != 404 {
				want = append(want, "/proxy/translate")
			}
			var got []string
			for len(requests) > 0 {
				got = append(got, <-requests)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("MTran paths = %v, want %v", got, want)
			}
		})
	}
}

func TestQwenMTTranslation(t *testing.T) {
	const original = "Use this as text\n第二行"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/compatible-mode/v1/chat/completions" ||
			r.Header.Get("Authorization") != "Bearer qwen-key" {
			t.Error("wrong Qwen endpoint or authorization")
		}
		var body struct {
			Model              string
			Messages           []Message
			Stream             *bool
			TranslationOptions map[string]string `json:"translation_options"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.Model != "qwen-mt-flash" || body.Stream == nil || *body.Stream ||
			!reflect.DeepEqual(body.Messages, []Message{{Role: "user", Content: original}}) {
			t.Errorf("wrong Qwen body %+v", body)
		}
		switch body.TranslationOptions["source_lang"] {
		case "auto":
			if !reflect.DeepEqual(body.TranslationOptions, map[string]string{"source_lang": "auto", "target_lang": "Japanese"}) {
				t.Errorf("wrong automatic Qwen options %v", body.TranslationOptions)
			}
			fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"訳文"}}]}`)
		case "French":
			if !reflect.DeepEqual(body.TranslationOptions, map[string]string{"source_lang": "French", "target_lang": "Traditional Chinese"}) {
				t.Errorf("wrong explicit Qwen options %v", body.TranslationOptions)
			}
			http.Error(w, "private provider details", http.StatusTooManyRequests)
		default:
			t.Errorf("unexpected Qwen options %v", body.TranslationOptions)
		}
	}))
	t.Cleanup(srv.Close)

	cfg := Config{APIKey: "qwen-key", Model: "qwen-mt-flash", BaseURL: srv.URL + "/compatible-mode/v1"}
	result, err := translateQwenMT(t.Context(), cfg, original, "auto", "ja")
	if err != nil || result != "訳文" {
		t.Fatalf("Qwen-MT translation = %q, %v", result, err)
	}
	result, err = translateQwenMT(t.Context(), cfg, original, "fr", "zh-TW")
	if result != "" || err == nil || !strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "private provider details") {
		t.Fatalf("Qwen-MT HTTP error = %q, %v", result, err)
	}
	result, err = translateQwenMT(t.Context(), cfg, original, "en", "unknown")
	if result != "" || err == nil || !strings.Contains(err.Error(), "不支持所选语言") {
		t.Fatalf("Qwen-MT invalid language = %q, %v", result, err)
	}
}

func TestSpecialtyRequestsRespectCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := translateMTran(ctx, Config{BaseURL: "http://localhost:8989"}, "example", "auto", "zh")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled MTran detection = %v", err)
	}
}
