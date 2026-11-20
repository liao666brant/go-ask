# 离线词典与会话存储

[项目索引](../../AGENTS.md) / `internal/data`

## 职责与接口

桌面应用通过 `Lookup(word) (Entry, bool)` 同步查词。词库嵌入程序，首次查询时建立行索引，之后按规范化词形二分查找。`Entry` 包含单词、音标、词性和中文释义。

应用启动时调用 `Open(path)` 打开或创建 SQLite 数据库，退出时调用 `Store.Close`。`NewConversation` 创建会话，`AddMessage` 追加消息，`Conversations` 搜索会话标题和正文，`Messages` 按写入顺序读取，`DeleteConversation` 删除会话及关联消息。

## 数据与依赖

数据库驱动为 `modernc.org/sqlite`。`conversations` 保存标题和更新时间，`messages` 以外键关联会话。追加消息会更新会话时间。英汉词库使用 ECDICT，生成脚本核验上游输入文件的 SHA-256。

## 测试与常见问题

运行 `go test ./internal/data`。测试使用临时数据库验证重开、检索及级联删除。遇到缺词时，先区分上游未收录与筛选规则排除；更新词库时按来源说明重新生成 TSV。

## 相关文件清单

- [词典查询](dictionary.go)、[SQLite 表与读写接口](history.go)
- [词库来源与生成方法](dictionary/SOURCE.md)、[生成脚本](dictionary/build.py)、[数据许可](dictionary/LICENSE)
- [词条和历史测试](data_test.go)
