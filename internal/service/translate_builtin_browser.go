package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

func translateMicrosoftBuiltIn(ctx context.Context, text, source, target string) (string, error) {
	// The current Edge endpoint accepts anonymous requests. The old /translate/auth endpoint returns 404.
	params := url.Values{
		"to": {chineseLanguage(target, "zh-Hans", "zh-Hant")}, "isEnterpriseClient": {"false"},
	}
	if source != "" && source != "auto" {
		params.Set("from", chineseLanguage(source, "zh-Hans", "zh-Hant"))
	}
	endpoint := "https://edge.microsoft.com/translate/translatetext?" + params.Encode()
	req, err := jsonRequest(ctx, endpoint, []string{text})
	if err != nil {
		return "", err
	}

	var data []struct {
		Translations []struct{ Text string }
	}
	if err := readClassicJSON(req, &data); err != nil {
		return "", err
	}
	if len(data) == 0 || len(data[0].Translations) == 0 || strings.TrimSpace(data[0].Translations[0].Text) == "" {
		return "", errors.New("微软内置翻译未返回译文")
	}
	return data[0].Translations[0].Text, nil
}

func translateGoogleBuiltIn(ctx context.Context, text, source, target string) (string, error) {
	if source == "" {
		source = "auto"
	}
	params := url.Values{
		"client": {"gtx"}, "dt": {"t"}, "dj": {"1"}, "ie": {"UTF-8"}, "oe": {"UTF-8"},
		"sl": {strings.ToLower(chineseLanguage(source, "zh-CN", "zh-TW"))},
		"tl": {strings.ToLower(chineseLanguage(target, "zh-CN", "zh-TW"))},
		"q":  {text},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://translate.google.com/translate_a/single?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	var data struct {
		Sentences []struct{ Trans string }
	}
	if err := readClassicJSON(req, &data); err != nil {
		// net/http includes the whole request URL in a transport error. Here it contains the source text.
		var transportError *url.Error
		if errors.As(err, &transportError) {
			return "", transportError.Err
		}
		return "", err
	}
	var result strings.Builder
	for _, sentence := range data.Sentences {
		result.WriteString(sentence.Trans)
	}
	if strings.TrimSpace(result.String()) == "" {
		return "", errors.New("谷歌内置翻译未返回译文")
	}
	return result.String(), nil
}
