# 翻译与问答服务

[项目索引](../../AGENTS.md) / `internal/service`

## 职责与接口

[桌面应用](../app/AGENTS.md) 传入 `Config` 和可取消的上下文。`Translate(ctx, cfg, provider, text, source, target)` 为单个来源返回译文或错误，多源并发由调用方组织。`Chat(ctx, cfg, messages, onDelta)` 接收当前会话的角色和正文，通过回调返回文本片段。

## 依赖与配置

HTTP 请求使用 Go 标准库。`Providers` 为设置界面和请求路由提供稳定服务 ID、用途及所需字段。DeepL 按密钥选择 Free 或 Pro 端点，百度和有道使用签名，Azure、Google、Yandex、腾讯混元、MTranServer 和 Qwen-MT 使用各自的翻译协议。OpenAI 兼容、智谱和 DeepSeek 共用 Chat Completions 处理，每家使用独立的地址、模型和凭据。`ChatEndpoint` 接受基础地址或完整的 Chat Completions 地址。

`BuiltIn` 标识无需用户配置的在线翻译来源。微软、谷歌、Yandex、腾讯 Transmart 和金山的内置适配使用各自的网页或客户端接口，与对应云 API 使用独立 ID。

## 测试与常见问题

运行 `go test ./internal/service`，本地模拟服务覆盖鉴权请求、取消、断流及服务错误。兼容端点可能在请求流式输出时返回普通 JSON；问答实现同时处理这种响应和 SSE。修改解析时保留中途断开与正常结束的区别。

## 相关文件清单

- [配置、机器翻译与 AI 翻译请求](translate.go)
- [服务目录](providers.go)、[云翻译适配](translate_classic.go)、[专用翻译适配](translate_specialty.go)
- 内置翻译适配见 [微软与谷歌](translate_builtin_browser.go)、[Yandex 与腾讯](translate_builtin_direct.go)、[金山](translate_builtin_iciba.go)
- [Chat 请求、端点解析与流式响应](chat.go)
- [服务协议回归测试](service_test.go)
