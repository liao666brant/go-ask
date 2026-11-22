# Go Ask

Go Ask 是使用 Go 和 NSGo 开发的桌面查词与 AI 问答浮窗，支持 Windows x64 和 Linux x64。

## 开始使用

解压对应平台的发布包，Windows 双击 `go-ask.exe`，Linux 运行 `./go-ask`。**启动后默认在后台，按 Ctrl+Q 显示窗口。** Linux 需要 GTK 4 运行库。Windows 发布包使用 GUI 子系统，启动时不会打开控制台。

| 操作 | 效果 |
| --- | --- |
| Ctrl+Q | 从当前应用唤起浮窗并聚焦输入框。有可读取的选区时直接查词或翻译，空选区时可输入或粘贴。 |
| Ctrl+Tab | 在同一个浮窗切换翻译和问答，各自保留当前输入。 |
| Enter / Shift+Enter | 提交当前输入 / 在输入框换行。 |
| Ctrl+N | 在问答模式开始新话题。 |
| Esc / 收起 | 隐藏浮窗，后台继续运行。已发送的回答继续生成。 |
| 设置或托盘 → 退出 | 完全退出并释放全局快捷键。 |

点浮窗右上角的设置图标，或右键托盘图标选择「设置」，即可打开独立设置窗口。重复打开会聚焦已有窗口，保留正在填写的内容。点击当前组合键，按下新组合并选择「应用快捷键」即可生效；注册失败时原键继续有效。

左键点击托盘图标可打开查词浮窗。右键菜单提供「打开 Go Ask」「设置」「关于 Go Ask」和「退出」。「关于 Go Ask」显示当前版本。

## 翻译和问答

- 单个已收录的英文词先显示本地英汉词条。内置 ECDICT 的 64,077 个词条，离线可用。「在线对照」可以再查看各来源的翻译。
- 在「设置 → 翻译服务 → 添加服务」中选择「免配置」来源，点击「启用翻译」即可使用，无需填写账号或密钥。内置来源使用在线接口，翻译时会向已启用的服务发送原文。
- 有凭据的服务可在对应表单中配置。问答可从「AI 提供商」单独选定 OpenAI 兼容、智谱或 DeepSeek。OpenAI 兼容配置支持填写本机服务地址。
- 提交句段会同时请求满足配置要求的已启用翻译来源。各来源独立显示，可复制、重试，或用「就此提问」将原文和译文带入新话题。
- 问答支持流式回答、停止生成、复制命令代码块和连续追问。点时钟图标，可按标题或消息内容搜索本地会话，重新打开后继续提问。
- 普通设置与 SQLite 历史保存在系统用户配置目录下的 `go-ask` 子目录。密钥保存在 Windows 凭据管理器或 Linux Secret Service；Linux 保存密钥前需要有可用且已解锁的密钥环。

## 桌面取词

Windows 通过 UI Automation 读取当前控件的选区，并支持标准 Edit 编辑框。对于没有公开选区的控件，浮窗会等待手动输入或粘贴。

### Linux 桌面集成

X11 读取焦点窗口拥有的 PRIMARY 选区。对于使用隐藏窗口持有选区的应用（如 GTK），Go Ask 通过 AT-SPI 读取焦点控件或活动窗口中可见网页正文的选中文字。此路径需要系统 Python 3、PyGObject 和 Atspi 2.0 的 GI 类型库。无法确认选区时会打开空输入框。

Wayland 的全局快捷键由桌面的 GlobalShortcuts 门户分配，实际组合键以系统授权为准。无法读取选区时，可复制后手动粘贴。如果桌面缺少快捷键门户或托盘，从应用菜单或再次运行同一发布包可打开已在后台的实例。

要在应用菜单显示 Go Ask，把发布包的 `go-ask` 放到桌面会话 PATH 包含的目录，把 `dev.goask.GoAsk.desktop` 放到 `~/.local/share/applications/`，图标 `dev.goask.GoAsk.png` 放到 `~/.local/share/icons/hicolor/256x256/apps/`。

## 从源码构建

需要 Go 1.27.3 或更高兼容版本及 Python 3。下列命令从项目根目录执行。

```sh
go test ./...
python3 tools/package.py
```

发布脚本为两个平台写入相同的 NSGo 应用 ID，启用后台单实例唤回。Linux 压缩包和 Windows ZIP 生成在 `dist/` 下，均含运行文件和依赖许可。开发启动可运行 `go run -ldflags="-X github.com/nsgo-dev/nsgo/core.appID=dev.goask.GoAsk" .`。

翻译交互参考 [STranslate](https://github.com/STranslate/STranslate)，原生界面使用 [NSGo v0.2.0](https://github.com/nsgo-dev/nsgo/releases/tag/v0.2.0)。词库数据取自 [ECDICT 固定版本](https://github.com/skywind3000/ECDICT/tree/39cfb9672c0906b63569e0453c2e6abfb9ab182a)。
