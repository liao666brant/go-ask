# Proposal

## Why

设置窗口把快捷键、面板和服务字段放在同一列，增加服务后会更难找到对应设置。用户需要配置多个翻译来源并比较结果，也需要为问答独立选择 AI 提供商。

## What Changes

- 设置窗口左侧切换「常规」「翻译服务」「AI 提供商」，右侧显示对应内容。常规沿用快捷键、失焦收起和空闲释放等操作。
- 增加按用途选择的服务目录和独立配置表单。翻译可同时启用多个已配置来源，问答使用一个明确选定的 AI 提供商。
- 参考 STranslate 的服务名单，采用各服务可用于第三方客户端的公开接口。接入范围和对应协议见 [技术设计](design.md)。
- 迁移现有 DeepL、百度和 OpenAI 兼容配置，保留已保存的历史、凭据和服务启用状态。

## Capabilities

### New Capabilities

- `settings-navigation` 管理设置窗口的分类、页面切换和编辑状态。
- `service-providers` 管理服务目录、配置、凭据和翻译及问答路由。

### Modified Capabilities

## Impact

影响 `internal/app` 的设置窗口、配置和请求调度，以及 `internal/service` 的接口适配。沿用系统凭据存储和本地会话历史。实现先在 Windows 验收，Linux 桌面适配与实测随本批功能统一进行。
