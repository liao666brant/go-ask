package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Public signing parameter shipped by iCIBA's anonymous browser translator.
const iCibaBuiltInSignatureSalt = "7ece94d9f9c202b0d2ec557dg4r9bc"

func translateICibaBuiltIn(ctx context.Context, text, source, target string) (string, error) {
	if source == "" {
		source = "auto"
	}
	const path = "/dictionary/fy/batch"
	const clientID, appID = "6", "1000006"
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	// The browser protocol signs parameters in the order client, key, timestamp.
	signature := md5.Sum([]byte(path + clientID + appID + timestamp + iCibaBuiltInSignatureSalt))
	query := url.Values{
		"client": {clientID}, "key": {appID}, "timestamp": {timestamp}, "signature": {fmt.Sprintf("%x", signature)},
	}
	body := struct {
		From     string   `json:"from"`
		To       string   `json:"to"`
		TextList []string `json:"textList"`
	}{chineseLanguage(source, "zh", "cht"), chineseLanguage(target, "zh", "cht"), []string{text}}
	req, err := jsonRequest(ctx, "https://dictionary.iciba.com"+path+"?"+query.Encode(), body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Origin", "https://www.iciba.com")
	req.Header.Set("Referer", "https://www.iciba.com/")
	req.Header.Set("User-Agent", "GoAsk")
	var data struct {
		Code json.RawMessage
		Data []json.RawMessage
	}
	if err := readClassicJSON(req, &data); err != nil {
		if transportErr, ok := errors.AsType[*url.Error](err); ok {
			return "", transportErr.Err
		}
		return "", err
	}
	if string(data.Code) == "1" || string(data.Code) == `"1"` {
		var translated []string
		for _, raw := range data.Data {
			var line string
			if err := json.Unmarshal(raw, &line); err != nil {
				var item struct {
					Output string `json:"out"`
				}
				if err := json.Unmarshal(raw, &item); err == nil {
					line = item.Output
				}
			}
			if strings.TrimSpace(line) != "" {
				translated = append(translated, line)
			}
		}
		if len(translated) != 0 {
			return strings.Join(translated, "\n"), nil
		}
	}
	return "", errors.New("金山内置翻译未返回译文")
}
