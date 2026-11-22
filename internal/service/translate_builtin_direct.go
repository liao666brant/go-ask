package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

var (
	yandexBuiltInID = func() string {
		id := builtInClientUUID()
		return hex.EncodeToString(id[:])
	}()
	transmartBuiltInID = func() string {
		id := builtInClientUUID()
		return fmt.Sprintf("browser-goask-1.0-%s-%x-%x-%x-%x-%x-%d",
			runtime.GOOS, id[:4], id[4:6], id[6:8], id[8:10], id[10:], time.Now().UnixMilli())
	}()
)

// Each service receives its own locally generated, process-scoped client ID.
func builtInClientUUID() [16]byte {
	var id [16]byte
	rand.Read(id[:])
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return id
}

func translateYandexBuiltIn(ctx context.Context, text, source, target string) (string, error) {
	lang := chineseLanguage(target, "zh", "zh")
	// Auto detection is requested by passing only the target language.
	if source != "" && source != "auto" {
		lang = chineseLanguage(source, "zh", "zh") + "-" + lang
	}
	query := url.Values{"ucid": {yandexBuiltInID}, "srv": {"android"}, "format": {"text"}}
	body := url.Values{"text": {text}, "lang": {lang}}
	const endpoint = "https://translate.yandex.net/api/v1/tr.json/translate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?"+query.Encode(), strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "GoAsk")
	var data struct {
		Code int
		Text []string
	}
	if err := readClassicJSON(req, &data); err != nil {
		return "", err
	}
	if data.Code != http.StatusOK || len(data.Text) == 0 || strings.TrimSpace(data.Text[0]) == "" {
		return "", errors.New("Yandex 内置翻译未返回译文")
	}
	return data.Text[0], nil
}

func translateTransmartBuiltIn(ctx context.Context, text, source, target string) (string, error) {
	if source == "" {
		source = "auto"
	}
	body := map[string]any{
		"header":         map[string]string{"fn": "auto_translation_block", "client_key": transmartBuiltInID},
		"type":           "plain",
		"model_category": "normal",
		"source":         map[string]string{"lang": chineseLanguage(source, "zh", "zh-TW"), "text_block": text},
		"target":         map[string]string{"lang": chineseLanguage(target, "zh", "zh-TW")},
	}
	req, err := jsonRequest(ctx, "https://transmart.qq.com/api/imt", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Referer", "https://yi.qq.com/zh-CN/index")
	req.Header.Set("User-Agent", "GoAsk")
	var data struct {
		Header struct {
			Result string `json:"ret_code"`
		}
		Translation string `json:"auto_translation"`
	}
	if err := readClassicJSON(req, &data); err != nil {
		return "", err
	}
	if data.Header.Result != "succ" || strings.TrimSpace(data.Translation) == "" {
		return "", errors.New("腾讯 Transmart 未返回译文")
	}
	return data.Translation, nil
}
