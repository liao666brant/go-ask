# 翻译与问答服务

[项目索引](../../AGENTS.md) / `internal/service`

## 职责与接口

[桌面应用](../app/AGENTS.md) 传入 `Config` 和可取消的上下文。`Translate(ctx, cfg, provider, text, source, target)` 为单个来源返回译文或错误，多源并发由调用方组织。`Chat(ctx, cfg, messages, onDelta)` 接收当前会话的角色和正文，通过回调返回文本片段。

## 依赖与配置

HTTP 请求使用 Go 标准库。DeepL 按密钥选择 Free 或 Pro 端点；百度使用 App ID 和密钥签名。AI 翻译与问答共用服务地址、模型和可选 API Key。`ChatEndpoint` 接受基础地址或完整的 Chat Completions 地址，供设置页与请求构造共用。

## 测试与常见问题

运行 `go test ./internal/service`，本地模拟服务覆盖鉴权请求、取消、断流及服务错误。兼容端点可能在请求流式输出时返回普通 JSON；问答实现同时处理这种响应和 SSE。修改解析时保留中途断开与正常结束的区别。

## 相关文件清单

- [配置、机器翻译与 AI 翻译请求](translate.go)
- [Chat 请求、端点解析与流式响应](chat.go)
- [服务协议回归测试](service_test.go)
