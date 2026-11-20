# 前台选区读取

[项目索引](../../AGENTS.md) / `internal/selection`

## 职责与接口

`Text(ctx context.Context) string` 是两个平台共用的调用入口。Go 按文件名后缀选择实现。[桌面宿主](../app/desktop.go) 在快捷键触发时先取选区，再聚焦查词框。空选区或无法读取时返回空字符串，浮窗等待输入或粘贴。

## 平台依赖与常见问题

- Windows 通过 UI Automation 的 TextPattern 读取焦点控件选区。标准 Edit 控件另有 Win32 消息读取路径，并检查焦点及密码属性。
- Linux X11 使用 `github.com/jezek/xgb` 读取 PRIMARY，并核对所有者和焦点是否属于同一顶层客户窗口。归属不明时，通过系统 Python 3 和 PyGObject 查询焦点进程活动窗口的 AT-SPI 文本选区。优先读取焦点控件，WebKit 正文可从唯一可见网页文档中读取；无法确认时返回空字符串。原生 Wayland 会话返回空字符串，快捷键门户由桌面宿主处理。

## 测试与质量

在隔离 X11 显示上运行 `GOASK_TEST_X11=1 go test -run TestX11SelectionWindow ./internal/selection` 验证窗口归属。桌面验收时，在编辑器窗口 A 选中一个词并按 `Ctrl+Q`，检查输入框和词条。再让 A 保留选区，切换到同一应用的无选区窗口 B，检查唤起后输入框为空。Linux 验收注明 X11 或 Wayland。

## 相关文件清单

- [Windows UI Automation 与 Edit 控件](selection_windows.go)
- [Linux 会话判断与 X11 选区](selection_linux.go)、[AT-SPI 焦点选区读取](focused_selection.py)、[X11 窗口归属测试](selection_linux_test.go)
