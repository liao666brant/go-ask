# Design

## Context

`internal/app/settings.go` 目前把所有设置渲染在一个滚动区域。翻译在 `internal/app/translation.go` 中并发调用 DeepL、百度和一个 AI 配置，问答与 AI 翻译共用该配置。`internal/app/preferences.go` 已采用双凭据项协调系统凭据和普通设置的写入。`internal/service` 已有可取消的 HTTP 客户端、Chat Completions 请求、JSON 与 SSE 响应解析。见 [提案](proposal.md)。

## Goals / Non-Goals

**Goals:** 为每个服务保存独立配置，复用现有多源翻译和流式问答流程，并继续使用已保存的本地会话与凭据。

**Non-Goals:** 本次范围为文本翻译与问答。STranslate 提供服务名称参考，界面沿用 Go Ask 的 NSGo 组件和样式。金山翻译和金山词霸暂缓。

## Decisions

### 设置页面

- 在现有设置窗口内实现左侧分类和右侧滚动内容。常规保留快捷键、失焦和空闲释放。翻译页列出专用翻译服务，AI 页列出通用模型服务、翻译启用项和默认问答提供商。
- 两个服务页共用服务选择入口和表单布局，按名称查找服务，按服务类型显示实际需要的字段。每种服务保留一份配置，用稳定 ID 关联目录、设置和结果卡。窗口作用域持有未保存的输入，切换分类或服务时保留。保存服务直接更新对应条目，避免覆盖已保存的其他分类。

### 服务目录与协议

| 服务 | 入口和用途 | 用户配置及接口依据 |
| --- | --- | --- |
| DeepL | 翻译，沿用 REST v2 的 Free 与 Pro 路由 | API Key。[DeepL Translate API](https://developers.deepl.com/api-reference/translate) |
| 百度翻译 | 翻译，沿用通用翻译 API | App ID、密钥。[百度通用翻译 API](https://fanyi-api.baidu.com/doc/21) |
| 有道翻译 | 翻译，文本翻译 API | 应用 ID、应用密钥，按 v3 规则签名。[有道文本翻译](https://ai.youdao.com/DOCSIRMA/html/trans/api/wbfy/index.html) |
| Azure Translator | 翻译，Text Translation v3 | 订阅 Key，资源要求时填写区域。[Azure 鉴权](https://learn.microsoft.com/en-us/azure/ai-services/translator/text-translation/reference/authentication) |
| Google Cloud Translation | 翻译，Basic v2 | Cloud API Key。[Google REST v2](https://docs.cloud.google.com/translate/docs/reference/rest/v2/translate) |
| Yandex Translate | 翻译，Yandex Cloud Translate v2 | 服务账号 API Key，或使用 IAM Token 并填写文件夹 ID。[Yandex Translate v2](https://aistudio.yandex.ru/en/docs/translate/api-ref/Translation/translate) |
| 腾讯混元翻译 | 翻译，TokenHub 专用翻译接口 | TokenHub API Key、翻译模型 ID。该接口是腾讯云公告推荐的迁移目标。[混元调用指南](https://cloud.tencent.com/document/product/1823/132252) · [文本翻译退市公告](https://cloud.tencent.cn/announce/detail/2482) |
| MTranServer | 翻译，自部署 HTTP 服务 | 服务地址、按部署填写 Bearer Token。使用当前公开接口 `/translate`，自动源语言调用 `/detect`，对缺少该接口的服务提示手动选择源语言。[MTranServer API](https://github.com/xxnuo/MTranServer/blob/main/docs/API_en.md#translation-endpoints) |
| Qwen-MT | 翻译，Model Studio 专用模型 | 按地域配置兼容端点、API Key、模型 ID。以单条用户消息和顶层 `translation_options` 提交，按文档映射语言。[Qwen-MT API](https://help.aliyun.com/zh/model-studio/qwen-mt-api) |
| OpenAI 兼容 | 翻译与问答，含现有自定义兼容服务 | 基础地址、模型 ID、API Key。沿用现有的无鉴权自托管配置。[Chat Completions](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create) |
| 智谱 | 翻译与问答，Chat Completions 预设 | 智谱端点、模型 ID、API Key。[智谱 OpenAI 兼容接口](https://docs.bigmodel.cn/cn/guide/develop/openai/introduction) |
| DeepSeek | 翻译与问答，Chat Completions 预设 | DeepSeek 端点、模型 ID、API Key。普通文本问答显示正文，支持推理分片先于正文到达。[Chat Completion](https://api-docs.deepseek.com/api/create-chat-completion/) · [思考模式](https://api-docs.deepseek.com/guides/thinking_mode/) |

金山翻译样例调用网页接口。金山词霸样例解析网页，历史开放文档仅定义单词查询，当前新用户可申请凭据这一点尚未确认。待获得可用开放接口和验证凭据后，再决定在线词典和文本翻译各自的接入范围。[STranslate 插件说明](https://github.com/STranslate/STranslate/blob/d718ee155988f17980c55308ca13419a1aefcd6d/src/docs/plugin-sdk-development.md) · [金山词霸查词文档](https://open.iciba.com/index.php?c=wiki&t=cc)

### 请求与存储

- 服务元数据描述名称、用途和所需字段，按服务 ID 分发请求。复用 `internal/service` 的 HTTP、取消和错误处理，有道签名、机器翻译请求及语言映射留在各自适配器。通用 AI 复用 Chat Completions，Qwen-MT 使用专用请求体。服务特有错误只显示在对应结果卡，显示的错误省略凭据。
- 翻译沿用查询编号、离线优先和独立重试。每次请求用对应服务的一份已保存配置。问答在发送时读取选定的提供商，正在生成的请求继续使用发起时的配置。新请求沿用现有会话上下文。
- 普通设置保存服务 ID、非秘密字段、翻译启用状态和问答选择。每个服务单独采用双凭据项，使用备用凭据写入成功后再切换普通设置中的活动项。Windows 凭据管理器限制单条凭据大小，按服务存储使各家的密钥长度互不影响。旧配置缺少服务列表时，在读取阶段把旧字段和当前凭据项映射给 DeepL、百度与 OpenAI 兼容。成功保存后切换到新格式，读取失败时保留旧配置供恢复。

## Risks / Trade-offs

- 各服务的语言代码、自动识别和结果格式不同。适配器为现有语言选项提供映射，用模拟响应验证支持、错误和单项重试。
- 云接口的模型和区域随服务商变化。通过预设提供接入入口，允许修改模型以及服务要求的区域或端点，验收记录使用过的配置。
- 需要用户凭据的服务以本地 HTTP 测试验证协议。真实服务联调记录实际测到的提供商，Windows 桌面验收可用隔离的模拟服务走完整设置和请求流程。
