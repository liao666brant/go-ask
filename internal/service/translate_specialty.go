package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func translateHunyuan(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	key, model := strings.TrimSpace(cfg.APIKey), strings.TrimSpace(cfg.Model)
	if key == "" || model == "" {
		return "", errors.New("请先填写腾讯混元翻译的 API Key 和模型")
	}
	body := struct {
		Model  string `json:"model"`
		Text   string `json:"text"`
		Source string `json:"source,omitempty"`
		Target string `json:"target"`
	}{Model: model, Text: text, Source: hunyuanLanguage(source), Target: hunyuanLanguage(target)}
	const endpoint = "https://tokenhub.tencentmaas.com/v1/api/translations"
	return translateCompletion(ctx, endpoint, key, body)
}

func hunyuanLanguage(language string) string {
	switch strings.ToLower(language) {
	case "auto":
		return ""
	case "zh-cn", "zh-hans":
		return "zh"
	case "zh-tw", "zh-hant", "zh-tr":
		return "zh-TR"
	default:
		return strings.ToLower(language)
	}
}

func translateMTran(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || base.Hostname() == "" || base.User != nil || base.ForceQuery ||
		base.RawQuery != "" || base.Fragment != "" || (base.Scheme != "http" && base.Scheme != "https") {
		return "", errors.New("请填写有效的 MTranServer 服务地址，例如 http://localhost:8989")
	}
	key := strings.TrimSpace(cfg.APIKey)
	if source == "" || source == "auto" {
		var detected struct {
			Language string `json:"language"`
		}
		err := postSpecialtyJSON(ctx, base.JoinPath("detect").String(), key, map[string]string{"text": text}, &detected)
		if err != nil {
			return "", fmt.Errorf("MTranServer 无法自动识别源语言，请手动选择源语言：%w", err)
		}
		source = strings.TrimSpace(detected.Language)
		if source == "" || source == "un" || source == "und" || source == "auto" {
			return "", errors.New("MTranServer 未识别出源语言，请手动选择源语言")
		}
	}
	body := struct {
		From string `json:"from"`
		To   string `json:"to"`
		Text string `json:"text"`
		HTML bool   `json:"html"`
	}{From: mtranLanguage(source), To: mtranLanguage(target), Text: text}
	var data struct {
		Result string `json:"result"`
	}
	if err := postSpecialtyJSON(ctx, base.JoinPath("translate").String(), key, body, &data); err != nil {
		return "", err
	}
	if strings.TrimSpace(data.Result) == "" {
		return "", errors.New("MTranServer 未返回译文")
	}
	return data.Result, nil
}

func mtranLanguage(language string) string {
	switch strings.ToLower(language) {
	case "zh", "zh-cn", "zh-hans":
		return "zh-Hans"
	case "zh-tw", "zh-hant":
		return "zh-Hant"
	default:
		return language
	}
}

func translateQwenMT(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	key, model := strings.TrimSpace(cfg.APIKey), strings.TrimSpace(cfg.Model)
	if key == "" || model == "" {
		return "", errors.New("请先填写 Qwen-MT 的 API Key 和模型")
	}
	endpoint, err := ChatEndpoint(cfg.BaseURL)
	if err != nil {
		return "", err
	}
	languages := map[string]string{
		"": "auto", "auto": "auto",
		"zh": "Chinese", "zh-cn": "Chinese", "zh-hans": "Chinese",
		"zh-tw": "Traditional Chinese", "zh-hant": "Traditional Chinese",
		"en": "English", "ja": "Japanese", "fr": "French", "de": "German",
	}
	from, to := languages[strings.ToLower(source)], languages[strings.ToLower(target)]
	if from == "" || to == "" || to == "auto" {
		return "", errors.New("Qwen-MT 不支持所选语言")
	}
	body := map[string]any{
		"model":               model,
		"messages":            []Message{{Role: "user", Content: text}},
		"translation_options": map[string]string{"source_lang": from, "target_lang": to},
		"stream":              false,
	}
	return translateCompletion(ctx, endpoint, key, body)
}

func translateCompletion(ctx context.Context, endpoint, key string, body any) (string, error) {
	var data completion
	if err := postSpecialtyJSON(ctx, endpoint, key, body, &data); err != nil {
		return "", err
	}
	if data.Error != nil || len(data.Choices) == 0 {
		return "", errors.New("翻译服务未返回译文")
	}
	choice := data.Choices[0]
	switch choice.Finish {
	case "sensitive", "content_filter":
		return "", errors.New("翻译服务未通过内容审核")
	case "length":
		return "", errors.New("译文超出服务长度限制，请缩短原文后重试")
	}
	if strings.TrimSpace(choice.Message.Content) == "" {
		return "", errors.New("翻译服务未返回译文")
	}
	return choice.Message.Content, nil
}

func postSpecialtyJSON(ctx context.Context, endpoint, key string, body, target any) error {
	req, err := jsonRequest(ctx, endpoint, body)
	if err != nil {
		return err
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}
