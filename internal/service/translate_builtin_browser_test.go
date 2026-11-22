package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestMicrosoftBuiltInProtocol(t *testing.T) {
	const original = "hello + world\nagain"
	for _, test := range []struct {
		name, source, target, query, response, want string
	}{
		{"auto", "auto", "zh", "isEnterpriseClient=false&to=zh-Hans",
			`[{"detectedLanguage":{"language":"en"},"translations":[{"text":"第一行\n第二行","to":"zh-Hans"}]}]`,
			"第一行\n第二行"},
		{"explicit", "zh-TW", "en", "from=zh-Hant&isEnterpriseClient=false&to=en",
			`[{"translations":[{"text":"First line.","to":"en","sentLen":{"srcSentLen":[7],"transSentLen":[11]}}]}]`,
			"First line."},
		{"empty", "", "zh", "isEnterpriseClient=false&to=zh-Hans", `[{"translations":[]}]`, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Host != "edge.microsoft.com" || r.URL.Path != "/translate/translatetext" || r.Method != http.MethodPost ||
					r.URL.RawQuery != test.query || r.Header.Get("Content-Type") != "application/json" ||
					r.Header.Get("Authorization") != "" || r.Header.Get("X-MT-Signature") != "" {
					t.Error("unexpected Microsoft translation request")
				}
				var body []string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, []string{original}) {
					t.Errorf("wrong translation payload: %v, %v", body, err)
				}
				fmt.Fprint(w, test.response)
			})
			result, err := translateMicrosoftBuiltIn(t.Context(), original, test.source, test.target)
			if result != test.want || (err != nil) != (test.want == "") {
				t.Fatalf("translation = %q, %v. Want %q", result, err, test.want)
			}
		})
	}
}

func TestGoogleBuiltInProtocol(t *testing.T) {
	const original = "你好 + 世界\n再见"
	for _, test := range []struct {
		source, target, sl, tl, response, want string
	}{
		{"", "zh", "auto", "zh-cn", `{"sentences":[{"trans":"第一行\n"},{"trans":null},{"trans":"第二行"}]}`, "第一行\n第二行"},
		{"zh-Hant", "JA", "zh-tw", "ja", `{"sentences":[{"trans":" "} ]}`, ""},
	} {
		t.Run(test.tl, func(t *testing.T) {
			mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				query := url.Values{"client": {"gtx"}, "dt": {"t"}, "dj": {"1"}, "ie": {"UTF-8"}, "oe": {"UTF-8"},
					"sl": {test.sl}, "tl": {test.tl}, "q": {original}}
				if r.Host != "translate.google.com" || r.URL.Path != "/translate_a/single" || r.Method != http.MethodGet ||
					!reflect.DeepEqual(r.URL.Query(), query) || r.Header.Get("Authorization") != "" || r.Header.Get("X-Goog-Api-Key") != "" {
					t.Error("unexpected Google translation request")
				}
				fmt.Fprint(w, test.response)
			})
			result, err := translateGoogleBuiltIn(t.Context(), original, test.source, test.target)
			if result != test.want || (err != nil) != (test.want == "") {
				t.Fatalf("translation = %q, %v. Want %q", result, err, test.want)
			}
		})
	}
}

func TestGoogleBuiltInHidesOriginalInTransportError(t *testing.T) {
	prior := client
	failure := errors.New("network unavailable")
	testClient := *client
	testClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, failure
	})
	client = &testClient
	t.Cleanup(func() { client = prior })
	const original = "private text"
	_, err := translateGoogleBuiltIn(t.Context(), original, "en", "zh")
	if !errors.Is(err, failure) || strings.Contains(err.Error(), url.QueryEscape(original)) {
		t.Fatalf("transport error was not sanitized: %v", err)
	}
}
