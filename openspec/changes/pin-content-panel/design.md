# Design

## Context

`internal/app/desktop.go` 用 `desktop.OpenPanel` 创建可获焦面板，并分别通过 `Focus` 和 `Hide` 唤起与收起。收起达到设置中的空闲时限后，宿主销毁面板，下次唤起时重新创建。`internal/app/window.go` 渲染翻译、问答和历史共用的标题栏。需求见 [提案](proposal.md) 和 [内容面板规范](specs/content-panel/spec.md)。

## Goals / Non-Goals

**Goals:** 在同一个面板实例中切换系统窗口层级，保持标题栏状态与窗口层级请求一致。

**Non-Goals:** 置顶状态的作用域限于内容面板与本次应用运行。

## Decisions

- 保留 `OpenPanel` 及原有显示和隐藏路径，用面板内容作用域调用 `desktop.SetWindowLevel`。关闭时请求 `WindowLevelNormal`，开启时请求 `WindowLevelFloating`。普通 `OpenWindow` 能承载窗口层级设置，但会改变现有面板的无边框和唤起行为。
- 用应用作用域的布尔 signal 保存状态，初始值为关闭。面板读取该状态来驱动自身的窗口层级，并更新标题栏按钮的操作名称和选中样式。每次重建面板时读取本次运行的置顶状态。
- 通过面板作用域的 `desktop.CanSetWindowLevel` 控制按钮可用性。该能力取决于实际窗口和桌面会话，因此按 Windows 或 Linux 名称判断无法区分 X11 与 Wayland。不可用说明应可见，并提供给辅助技术。

## Risks / Trade-offs

- X11 由窗口管理器执行置顶请求，撤回并重新映射后由 NSGo 重新请求层级。以 Windows 和受支持的 X11 桌面上的真实窗口遮挡测试验证切换与重新唤起，再在 Wayland 验证按钮的不可用说明。依据为 [NSGo v0.2.0 的窗口层级接口](https://github.com/nsgo-dev/nsgo/blob/v0.2.0/desktop/desktop.go) 和 [X11 窗口层级实现](https://github.com/nsgo-dev/nsgo/blob/v0.2.0/internal/platform/linux/level.go)。
