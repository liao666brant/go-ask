package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
)

func Chat(ctx context.Context, cfg Config, messages []Message, onDelta func(string)) error {
	withPrompt := make([]Message, 0, len(messages)+1)
	withPrompt = append(withPrompt, Message{"system", "你是桌面快速问答助手。默认用简体中文，先给直接答案，再作必要解释。命令示例使用带语言的 Markdown 代码块。"})
	withPrompt = append(withPrompt, messages...)
	return complete(ctx, cfg, withPrompt, true, onDelta)
}

// ChatEndpoint 接受兼容服务的基础地址，也接受完整的 Chat Completions 地址。
func ChatEndpoint(base string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.Hostname() == "" || u.User != nil || u.ForceQuery || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("请填写有效的 AI 服务基础地址，例如 https://api.openai.com/v1")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", errors.New("AI 服务地址应以 https:// 或 http:// 开头")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/chat/completions") {
		u.Path += "/chat/completions"
	}
	return u.String(), nil
}

type completion struct {
	Choices []struct {
		Message Message
		Delta   Message
		Finish  string `json:"finish_reason"`
	}
	Error *json.RawMessage `json:"error"`
}

func complete(ctx context.Context, cfg Config, messages []Message, stream bool, onDelta func(string)) error {
	endpoint, err := ChatEndpoint(cfg.BaseURL)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return errors.New("请先填写 AI 模型名称")
	}
	req, err := jsonRequest(ctx, endpoint, map[string]any{"model": model, "messages": messages, "stream": stream})
	if err != nil {
		return err
	}
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if stream {
		req.Header.Set("Accept", "text/event-stream, application/json")
	}
	resp, err := send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		return readStream(resp.Body, onDelta)
	}
	var data completion
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	if data.Error != nil || len(data.Choices) == 0 || data.Choices[0].Message.Content == "" {
		return errors.New("AI 服务未返回回答，请检查模型配置")
	}
	onDelta(data.Choices[0].Message.Content)
	return nil
}

func readStream(body io.Reader, onDelta func(string)) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	var data []string
	var received, finished bool
	dispatch := func() (bool, error) {
		if len(data) == 0 {
			return false, nil
		}
		raw := strings.Join(data, "\n")
		data = data[:0]
		if strings.TrimSpace(raw) == "[DONE]" {
			return true, nil
		}
		var chunk completion
		if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
			return false, err
		}
		if chunk.Error != nil {
			return false, errors.New("AI 服务中断了回答")
		}
		if len(chunk.Choices) > 0 {
			c := chunk.Choices[0]
			if c.Delta.Content != "" {
				onDelta(c.Delta.Content)
				received = true
			}
			finished = finished || c.Finish != ""
		}
		return false, nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			done, err := dispatch()
			if err != nil {
				return err
			}
			if done {
				finished = true
				break
			}
		} else if value, ok := strings.CutPrefix(line, "data:"); ok {
			data = append(data, strings.TrimPrefix(value, " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	done, err := dispatch()
	if err != nil {
		return err
	}
	if !done && !finished {
		return io.ErrUnexpectedEOF
	}
	if !received {
		return errors.New("AI 服务未返回回答正文")
	}
	return nil
}
