package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestClassicCloudTranslation(t *testing.T) {
	const text = "hello + world\nagain"
	for _, tc := range []struct {
		name, host, path  string
		headers           map[string]string
		query             string
		request, response string
		source, target    string
		cfg               Config
		translate         func(context.Context, Config, string, string, string) (string, error)
	}{
		{
			name: "Azure", host: "api.cognitive.microsofttranslator.com", path: "/translate",
			headers: map[string]string{"Ocp-Apim-Subscription-Key": "azure-key", "Ocp-Apim-Subscription-Region": "eastus"},
			query:   "api-version=3.0&to=zh-Hans",
			request: `[{"Text":"hello + world\nagain"}]`, response: `[{"translations":[{"text":"译文","to":"zh-Hans"}]}]`,
			source: "auto", target: "zh",
			cfg: Config{APIKey: "azure-key", Region: "eastus"}, translate: translateAzure,
		},
		{
			name: "Azure explicit language", host: "api.cognitive.microsofttranslator.com", path: "/translate",
			headers: map[string]string{"Ocp-Apim-Subscription-Key": "azure-key", "Ocp-Apim-Subscription-Region": ""},
			query:   "api-version=3.0&from=zh-Hant&to=en",
			request: `[{"Text":"hello + world\nagain"}]`, response: `[{"translations":[{"text":"译文","to":"en"}]}]`,
			source: "zh-TW", target: "en",
			cfg: Config{APIKey: "azure-key"}, translate: translateAzure,
		},
		{
			name: "Google", host: "translation.googleapis.com", path: "/language/translate/v2",
			headers:  map[string]string{"X-Goog-Api-Key": "google-key"},
			request:  `{"q":["hello + world\nagain"],"target":"zh-CN","format":"text"}`,
			response: `{"data":{"translations":[{"translatedText":"译文","detectedSourceLanguage":"en"}]}}`,
			source:   "auto", target: "zh",
			cfg: Config{APIKey: "google-key"}, translate: translateGoogle,
		},
		{
			name: "Yandex service account", host: "translate.api.cloud.yandex.net", path: "/translate/v2/translate",
			headers:  map[string]string{"Authorization": "Api-Key yandex-key"},
			request:  `{"texts":["hello + world\nagain"],"targetLanguageCode":"zh","format":"PLAIN_TEXT"}`,
			response: `{"translations":[{"text":"译文"}]}`,
			source:   "auto", target: "zh-CN",
			cfg: Config{APIKey: "yandex-key"}, translate: translateYandex,
		},
		{
			name: "Yandex IAM", host: "translate.api.cloud.yandex.net", path: "/translate/v2/translate",
			headers:  map[string]string{"Authorization": "Bearer yandex-token"},
			request:  `{"texts":["hello + world\nagain"],"sourceLanguageCode":"zh","targetLanguageCode":"en","format":"PLAIN_TEXT","folderId":"cloud-folder"}`,
			response: `{"translations":[{"text":"译文"}]}`,
			source:   "zh-Hans", target: "en",
			cfg: Config{APIKey: "yandex-token", FolderID: "cloud-folder"}, translate: translateYandex,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Host != tc.host || r.URL.Path != tc.path || r.URL.RawQuery != tc.query ||
					r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected request: %s %s%s content-type %q", r.Method, r.Host, r.URL, r.Header.Get("Content-Type"))
				}
				for name, value := range tc.headers {
					if got := r.Header.Get(name); got != value {
						t.Errorf("header %s = %q, want %q", name, got, value)
					}
				}
				var got, want any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				if err := json.Unmarshal([]byte(tc.request), &want); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("request payload = %#v, want %#v", got, want)
				}
				fmt.Fprint(w, tc.response)
			})
			result, err := tc.translate(context.Background(), tc.cfg, text, tc.source, tc.target)
			if err != nil || result != "译文" {
				t.Fatalf("translation = %q, %v", result, err)
			}
		})
	}
}

func TestYoudaoSignatureAndLanguage(t *testing.T) {
	for _, tc := range []struct{ text, signatureInput, source, from, target, to string }{
		{"short + 输入", "short + 输入", "", "auto", "zh", "zh-CHS"},
		{"一二三四五六七八九十中甲乙丙丁戊己庚辛壬癸", "一二三四五六七八九十21甲乙丙丁戊己庚辛壬癸", "zh-TW", "zh-CHT", "ja", "ja"},
	} {
		t.Run(tc.from, func(t *testing.T) {
			mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				if r.Method != http.MethodPost || r.Host != "openapi.youdao.com" || r.URL.Path != "/api" || r.URL.RawQuery != "" ||
					r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Errorf("unexpected request: %s %s%s content-type %q", r.Method, r.Host, r.URL, r.Header.Get("Content-Type"))
				}
				salt, timestamp := r.PostForm.Get("salt"), r.PostForm.Get("curtime")
				seconds, err := strconv.ParseInt(timestamp, 10, 64)
				if err != nil || time.Since(time.Unix(seconds, 0)).Abs() > time.Minute || salt == "" {
					t.Errorf("invalid timestamp or salt: %q %q", timestamp, salt)
				}
				sign := sha256.Sum256([]byte("app" + tc.signatureInput + salt + timestamp + "secret"))
				want := map[string]string{
					"q": tc.text, "from": tc.from, "to": tc.to, "appKey": "app",
					"strict": "true", "signType": "v3", "sign": fmt.Sprintf("%x", sign),
				}
				for key, value := range want {
					if got := r.PostForm.Get(key); got != value {
						t.Errorf("form %s = %q, want %q", key, got, value)
					}
				}
				fmt.Fprint(w, `{"errorCode":"0","translation":["第一行","第二行"]}`)
			})
			result, err := translateYoudao(context.Background(), Config{AppID: "app", AppSecret: "secret"}, tc.text, tc.source, tc.target)
			if err != nil || result != "第一行\n第二行" {
				t.Fatalf("translation = %q, %v", result, err)
			}
		})
	}
}

func TestClassicTranslationErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		response  string
		want      string
		translate func(context.Context, Config, string, string, string) (string, error)
	}{
		{"Azure quota", http.StatusTooManyRequests, `{"error":{"message":"hidden-private-data"}}`, "HTTP 429", translateAzure},
		{"Google credentials", http.StatusForbidden, `{"error":{"message":"hidden-private-data"}}`, "HTTP 403", translateGoogle},
		{"Yandex empty", http.StatusOK, `{"translations":[]}`, "未返回译文", translateYandex},
		{"Youdao service error", http.StatusOK, `{"errorCode":"108","errorMessage":"hidden-private-data"}`, "108", translateYoudao},
		{"Youdao invalid status", http.StatusOK, `{"errorCode":"hidden-private-data"}`, "无效的状态", translateYoudao},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.response)
			})
			cfg := Config{APIKey: "test-key", AppID: "app", AppSecret: "secret"}
			result, err := tc.translate(context.Background(), cfg, "hello", "en", "zh")
			if result != "" || err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "hidden-private-data") {
				t.Fatalf("translation error = %q, %v", result, err)
			}
		})
	}
}

func mockClassicEndpoint(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	prior := client
	testClient := *prior
	testClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.Host = req.URL.Host
		req.URL.Host, req.URL.Scheme = endpoint.Host, endpoint.Scheme
		return http.DefaultTransport.RoundTrip(req)
	})
	client = &testClient
	t.Cleanup(func() { client = prior })
}
