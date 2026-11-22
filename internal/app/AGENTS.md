# 桌面应用

[项目索引](../../AGENTS.md) / `internal/app`

## 职责与入口

`Run(trayIcon []byte) error` 是根入口调用的桌面接口。它先读取用户配置、打开历史数据库，再启动 NSGo。桌面宿主管理托盘、全局热键和窗口；翻译、问答和历史视图共享应用状态。

## 数据与依赖

普通选项写入用户配置目录下的 `go-ask/settings.json`，历史使用同目录的 `history.db`，服务密钥由 NSGo 的系统凭据存储读写。宿主在显示浮窗前调用 [selection](../selection/AGENTS.md) 取词。翻译流程先查 [data](../data/AGENTS.md) 的离线词库，在线请求交给 [service](../service/AGENTS.md)。后台响应经 `Scope.Post` 更新界面，翻译请求编号用于丢弃旧结果。

设置左侧分为常规、翻译服务和 AI 提供商。应用按服务 ID 保存设置与凭据，先写入该服务的备用凭据项，再随设置文件切换服务地址和凭据。旧格式按原活动凭据项读取，并在成功保存服务时迁移。免配置来源只保存启用状态。各翻译来源独立启用，问答使用明确选定的 AI 提供商。词条上的「在线对照」使用已显示词条的原文和语言。流式问答失败时，错误在本次运行期间随对应会话显示。

## 测试与常见问题

运行 `go test ./internal/app`。工作流测试覆盖多源翻译单独重试、连续追问及重启恢复。按 `Ctrl+Q` 唤起内容面板。收起到达设置中的空闲时限后，桌面宿主销毁面板，下次唤起时重建。置顶和问答状态由应用作用域持有。设置和关于窗口已打开时，重复打开会复用对应窗口。

## 相关文件清单

- [启动](run.go)、[桌面宿主](desktop.go)、[应用状态](application.go)
- [设置和关于窗口](settings.go)、[配置与凭据](preferences.go)
- [服务目录与配置界面](settings_providers.go)、[设置导航与服务流程测试](settings_navigation_test.go)
- [凭据与设置保存失败回归](preferences_test.go)
- [查词浮窗](window.go)、[翻译流程](translation.go)
- [问答流程](chat.go)、[会话与历史界面](chat_view.go)
- [窗口及热键测试](desktop_test.go)、[多源翻译测试](translation_workflow_test.go)、[问答工作流测试](workflows_test.go)
