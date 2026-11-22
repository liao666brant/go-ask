package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func translateAzure(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	key := strings.TrimSpace(cfg.APIKey)
	if key == "" {
		return "", errors.New("请先填写 Azure Translator 订阅密钥")
	}
	params := url.Values{"api-version": {"3.0"}, "to": {chineseLanguage(target, "zh-Hans", "zh-Hant")}}
	if source != "" && source != "auto" {
		params.Set("from", chineseLanguage(source, "zh-Hans", "zh-Hant"))
	}
	req, err := jsonRequest(ctx, "https://api.cognitive.microsofttranslator.com/translate?"+params.Encode(), []map[string]string{{"Text": text}})
	if err != nil {
		return "", err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", key)
	if region := strings.TrimSpace(cfg.Region); region != "" {
		req.Header.Set("Ocp-Apim-Subscription-Region", region)
	}
	var data []struct{ Translations []struct{ Text string } }
	if err := readClassicJSON(req, &data); err != nil {
		return "", err
	}
	if len(data) == 0 || len(data[0].Translations) == 0 || data[0].Translations[0].Text == "" {
		return "", errors.New("Azure Translator 未返回译文")
	}
	return data[0].Translations[0].Text, nil
}

func translateGoogle(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	key := strings.TrimSpace(cfg.APIKey)
	if key == "" {
		return "", errors.New("请先填写 Google Cloud Translation API Key")
	}
	body := map[string]any{"q": []string{text}, "target": chineseLanguage(target, "zh-CN", "zh-TW"), "format": "text"}
	if source != "" && source != "auto" {
		body["source"] = chineseLanguage(source, "zh-CN", "zh-TW")
	}
	req, err := jsonRequest(ctx, "https://translation.googleapis.com/language/translate/v2", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Goog-Api-Key", key)
	var data struct {
		Data struct {
			Translations []struct{ TranslatedText string }
		}
	}
	if err := readClassicJSON(req, &data); err != nil {
		return "", err
	}
	if len(data.Data.Translations) == 0 || data.Data.Translations[0].TranslatedText == "" {
		return "", errors.New("Google Cloud Translation 未返回译文")
	}
	return data.Data.Translations[0].TranslatedText, nil
}

func translateYandex(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	key := strings.TrimSpace(cfg.APIKey)
	if key == "" {
		return "", errors.New("请先填写 Yandex Translate API Key 或 IAM Token")
	}
	body := map[string]any{"texts": []string{text}, "targetLanguageCode": chineseLanguage(target, "zh", target), "format": "PLAIN_TEXT"}
	if source != "" && source != "auto" {
		body["sourceLanguageCode"] = chineseLanguage(source, "zh", source)
	}
	auth := "Api-Key " + key
	if folderID := strings.TrimSpace(cfg.FolderID); folderID != "" {
		body["folderId"] = folderID
		auth = "Bearer " + key
	}
	req, err := jsonRequest(ctx, "https://translate.api.cloud.yandex.net/translate/v2/translate", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", auth)
	var data struct{ Translations []struct{ Text string } }
	if err := readClassicJSON(req, &data); err != nil {
		return "", err
	}
	if len(data.Translations) == 0 || data.Translations[0].Text == "" {
		return "", errors.New("Yandex Translate 未返回译文")
	}
	return data.Translations[0].Text, nil
}

func translateYoudao(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	appID, secret := strings.TrimSpace(cfg.AppID), strings.TrimSpace(cfg.AppSecret)
	if appID == "" || secret == "" {
		return "", errors.New("请先填写有道翻译应用 ID 和应用密钥")
	}
	if source == "" {
		source = "auto"
	}
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes)
	now := strconv.FormatInt(time.Now().Unix(), 10)
	input := text
	chars := []rune(text)
	if len(chars) > 20 {
		input = string(chars[:10]) + strconv.Itoa(len(chars)) + string(chars[len(chars)-10:])
	}
	signature := sha256.Sum256([]byte(appID + input + salt + now + secret))
	params := url.Values{
		"q": {text}, "from": {chineseLanguage(source, "zh-CHS", "zh-CHT")}, "to": {chineseLanguage(target, "zh-CHS", "zh-CHT")},
		"appKey": {appID}, "salt": {salt}, "sign": {hex.EncodeToString(signature[:])}, "signType": {"v3"}, "curtime": {now},
		"strict": {"true"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openapi.youdao.com/api", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var data struct {
		ErrorCode   string
		Translation []string
	}
	if err := readClassicJSON(req, &data); err != nil {
		return "", err
	}
	code, err := strconv.Atoi(data.ErrorCode)
	if err != nil {
		return "", errors.New("有道翻译返回了无效的状态")
	}
	if code != 0 {
		return "", fmt.Errorf("有道翻译错误 %d", code)
	}
	result := strings.Join(data.Translation, "\n")
	if result == "" {
		return "", errors.New("有道翻译未返回译文")
	}
	return result, nil
}

func readClassicJSON(req *http.Request, data any) error {
	resp, err := send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(data)
}

func chineseLanguage(code, simplified, traditional string) string {
	switch strings.ToLower(code) {
	case "zh", "zh-cn", "zh-hans":
		return simplified
	case "zh-tw", "zh-hant":
		return traditional
	default:
		return code
	}
}
