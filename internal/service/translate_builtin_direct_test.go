package service

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"testing"
)

func TestYandexBuiltIn(t *testing.T) {
	const text = "hello + world\n第二行"
	for _, tc := range []struct {
		name, source, target, lang, response, want string
	}{
		{"automatic", "auto", "zh", "zh", `{"code":200,"text":["你好"]}`, "你好"},
		{"explicit", "zh-TW", "fr", "zh-fr", `{"code":200,"text":["Bonjour"]}`, "Bonjour"},
		{"empty result", "en", "zh-Hans", "en-zh", `{"code":200,"text":[]}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				clientID, err := hex.DecodeString(query.Get("ucid"))
				if r.Method != http.MethodPost || r.Host != "translate.yandex.net" || r.URL.Path != "/api/v1/tr.json/translate" ||
					r.Header.Get("Authorization") != "" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" ||
					query.Get("srv") != "android" || query.Get("format") != "text" || err != nil || len(clientID) != 16 {
					t.Error("wrong Yandex built-in request")
				}
				if err := r.ParseForm(); err != nil || r.PostForm.Get("text") != text || r.PostForm.Get("lang") != tc.lang {
					t.Errorf("wrong Yandex form: %v, %v", r.PostForm, err)
				}
				fmt.Fprint(w, tc.response)
			})
			got, err := translateYandexBuiltIn(t.Context(), text, tc.source, tc.target)
			if got != tc.want || (tc.want == "" && (err == nil || !strings.Contains(err.Error(), "未返回译文"))) ||
				(tc.want != "" && err != nil) {
				t.Fatalf("translation = %q, %v", got, err)
			}
		})
	}
}

func TestTransmartBuiltIn(t *testing.T) {
	const text = "hello + world\n第二行"
	mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Host != "transmart.qq.com" || r.URL.Path != "/api/imt" ||
			r.Header.Get("Authorization") != "" || r.Header.Get("Content-Type") != "application/json" ||
			r.Header.Get("Referer") != "https://yi.qq.com/zh-CN/index" {
			t.Error("wrong Transmart built-in request")
		}
		var body struct {
			Header        map[string]string
			Type          string
			ModelCategory string `json:"model_category"`
			Source        map[string]string
			Target        map[string]string
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !strings.HasPrefix(body.Header["client_key"], "browser-goask-1.0-"+runtime.GOOS+"-") ||
			body.Header["fn"] != "auto_translation_block" || body.Type != "plain" || body.ModelCategory != "normal" ||
			body.Source["lang"] != "auto" || body.Source["text_block"] != text || body.Target["lang"] != "zh-TW" {
			t.Error("wrong Transmart built-in payload")
		}
		fmt.Fprint(w, `{"header":{"ret_code":"succ"},"auto_translation":"早安"}`)
	})
	got, err := translateTransmartBuiltIn(t.Context(), text, "", "zh-Hant")
	if err != nil || got != "早安" {
		t.Fatalf("translation = %q, %v", got, err)
	}
}

func TestTransmartBuiltInServiceError(t *testing.T) {
	mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"header":{"ret_code":"service-error-details"}}`)
	})
	got, err := translateTransmartBuiltIn(t.Context(), "example", "en", "zh")
	if got != "" || err == nil || !strings.Contains(err.Error(), "未返回译文") || strings.Contains(err.Error(), "service-error-details") {
		t.Fatalf("translation = %q, %v", got, err)
	}
}
