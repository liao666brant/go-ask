# 离线英汉词库

数据来自 [ECDICT 的固定版本](https://github.com/skywind3000/ECDICT/tree/39cfb9672c0906b63569e0453c2e6abfb9ab182a)，附上游 [MIT 许可证](LICENSE)。本项目从 `ecdict.csv` 筛出带考试标签、收录于 BNC 或当代语料库词频表、包含计算机释义的英文单词，另补入少量常用技术词。当前内嵌 64,077 条。音标和释义沿用上游，词性标签提取自已有释义。

使用 Python 3.8 及以上版本运行 `python3 build.py /path/to/ecdict.csv` 可从固定版本重新生成 `entries.tsv`。生成器校验输入的 SHA-256。应用直接使用内嵌词库，安装和查询都无需下载数据。
