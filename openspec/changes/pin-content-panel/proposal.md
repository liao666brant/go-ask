# Proposal

## Why

用户在另一个应用里写作或阅读时，需要把 Go Ask 的翻译和问答内容留在前面，也需要能随时恢复普通窗口层级，避免面板挡住其他应用。

## What Changes

- 在内容面板标题栏加入置顶按钮，显示当前状态，点击即可切换。
- 翻译、问答和历史会话共用置顶状态。应用启动时默认关闭，本次运行内收起后再打开会保留上次选择。
- 当前桌面不支持调整窗口层级时，按钮显示不可用并说明原因。

## Capabilities

### New Capabilities

- `content-panel`: 内容面板的置顶控制、状态与桌面兼容行为。

### Modified Capabilities

## Impact

影响 `internal/app` 的内容面板标题栏、桌面宿主和窗口流程测试。窗口层级由 NSGo desktop 管理，验收覆盖 Windows 和 Linux 的相关桌面会话。
