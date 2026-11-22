package service

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestICibaBuiltIn(t *testing.T) {
	const text = "hello + world\n第二行"
	for _, tc := range []struct {
		name, response, want string
	}{
		{"numeric status and mixed data", `{"code":1,"data":[{"out":"第一行"},null,"第二行"]}`, "第一行\n第二行"},
		{"string status", `{"code":"1","data":["译文"]}`, "译文"},
		{"rejected status", `{"code":2,"data":["private-response-details"]}`, ""},
		{"no text", `{"code":1,"data":[{},null," " ]}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockClassicEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Host != "dictionary.iciba.com" || r.URL.Path != "/dictionary/fy/batch" ||
					r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" ||
					r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Origin") != "https://www.iciba.com" {
					t.Error("incorrect iCIBA built-in request")
				}
				query := r.URL.Query()
				timestamp := query.Get("timestamp")
				ms, err := strconv.ParseInt(timestamp, 10, 64)
				if err != nil || time.Since(time.UnixMilli(ms)).Abs() > time.Minute {
					t.Error("invalid iCIBA signing timestamp")
				}
				expectedSign := fmt.Sprintf("%x", md5.Sum([]byte(r.URL.Path+"6"+"1000006"+timestamp+iCibaBuiltInSignatureSalt)))
				if query.Get("signature") != expectedSign || query.Get("client") != "6" || query.Get("key") != "1000006" {
					t.Error("incorrect public browser signature")
				}
				var body struct {
					From, To string
					TextList []string
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.From != "auto" || body.To != "cht" ||
					len(body.TextList) != 1 || body.TextList[0] != text {
					t.Errorf("incorrect translation payload: %+v, %v", body, err)
				}
				fmt.Fprint(w, tc.response)
			})
			got, err := translateICibaBuiltIn(t.Context(), text, "", "zh-Hant")
			if got != tc.want {
				t.Fatalf("translation = %q, want %q", got, tc.want)
			}
			if tc.want != "" && err != nil {
				t.Fatal(err)
			}
			if tc.want == "" && (err == nil || !strings.Contains(err.Error(), "未返回译文") ||
				strings.Contains(err.Error(), "private-response-details")) {
				t.Fatalf("unexpected service error: %v", err)
			}
		})
	}
}
