package service

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DeepL = "DeepL"
	Baidu = "Baidu"
	AI    = "AI"
)

type Config struct {
	DeepLKey    string
	BaiduAppID  string
	BaiduSecret string
	BaseURL     string
	APIKey      string
	Model       string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var client = &http.Client{
	Timeout:       2 * time.Minute,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func Translate(ctx context.Context, cfg Config, provider, text, source, target string) (string, error) {
	if strings.TrimSpace(text) == "" || target == "" || target == "auto" {
		return "", errors.New("请填写原文并选择目标语言")
	}
	switch provider {
	case DeepL:
		return translateDeepL(ctx, cfg, text, source, target)
	case Baidu:
		return translateBaidu(ctx, cfg, text, source, target)
	case AI:
		prompt := "你是翻译器。将用户的全部文本翻译为 " + target + "。保留原文段落和格式，仅输出译文。"
		if source != "" && source != "auto" {
			prompt += "原文语言为 " + source + "。"
		}
		var result strings.Builder
		err := complete(ctx, cfg, []Message{{"system", prompt}, {"user", text}}, false, func(s string) { result.WriteString(s) })
		return result.String(), err
	default:
		return "", fmt.Errorf("未知翻译服务 %q", provider)
	}
}

func translateDeepL(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	key := strings.TrimSpace(cfg.DeepLKey)
	if key == "" {
		return "", errors.New("请先填写 DeepL 密钥")
	}
	host := "https://api.deepl.com"
	if strings.HasSuffix(key, ":fx") {
		host = "https://api-free.deepl.com"
	}
	lang := strings.ToUpper(target)
	switch lang {
	case "ZH", "ZH-CN":
		lang = "ZH-HANS"
	case "EN":
		lang = "EN-US"
	}
	body := map[string]any{"text": []string{text}, "target_lang": lang}
	if source != "" && source != "auto" {
		body["source_lang"] = strings.ToUpper(strings.Split(source, "-")[0])
	}
	req, err := jsonRequest(ctx, host+"/v2/translate", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "DeepL-Auth-Key "+key)
	resp, err := send(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data struct{ Translations []struct{ Text string } }
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Translations) == 0 || data.Translations[0].Text == "" {
		return "", errors.New("DeepL 未返回译文")
	}
	return data.Translations[0].Text, nil
}

func translateBaidu(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	if cfg.BaiduAppID == "" || cfg.BaiduSecret == "" {
		return "", errors.New("请先填写百度翻译 APP ID 和密钥")
	}
	var rawSalt [12]byte
	if _, err := rand.Read(rawSalt[:]); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(rawSalt[:])
	// 百度通用翻译按原文签名，随后再进行表单编码。
	sign := md5.Sum([]byte(cfg.BaiduAppID + text + salt + cfg.BaiduSecret))
	aliases := map[string]string{"": "auto", "zh-CN": "zh", "zh-TW": "cht", "ja": "jp", "ko": "kor", "fr": "fra", "es": "spa", "vi": "vie"}
	baiduLang := func(code string) string {
		if mapped, ok := aliases[code]; ok {
			return mapped
		}
		return code
	}
	params := url.Values{"appid": {cfg.BaiduAppID}, "q": {text}, "salt": {salt},
		"sign": {hex.EncodeToString(sign[:])}, "from": {baiduLang(source)}, "to": {baiduLang(target)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://fanyi-api.baidu.com/api/trans/vip/translate", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := send(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data struct {
		ErrorCode string                 `json:"error_code"`
		Results   []struct{ Dst string } `json:"trans_result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.ErrorCode != "" && data.ErrorCode != "52000" {
		return "", fmt.Errorf("百度翻译错误 %s", data.ErrorCode)
	}
	if len(data.Results) == 0 {
		return "", errors.New("百度翻译未返回译文")
	}
	lines := make([]string, 0, len(data.Results))
	for _, item := range data.Results {
		lines = append(lines, item.Dst)
	}
	return strings.Join(lines, "\n"), nil
}

func jsonRequest(ctx context.Context, endpoint string, body any) (*http.Request, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func send(req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("服务返回 HTTP %d（%s）", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp, nil
}
