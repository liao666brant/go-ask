package service

const (
	Azure    = "azure"
	Google   = "google"
	Yandex   = "yandex"
	Youdao   = "youdao"
	Hunyuan  = "hunyuan"
	MTran    = "mtranserver"
	QwenMT   = "qwen-mt"
	Zhipu    = "zhipu"
	DeepSeek = "deepseek"

	MicrosoftBuiltIn = "microsoft-builtin"
	GoogleBuiltIn    = "google-builtin"
	YandexBuiltIn    = "yandex-builtin"
	TransmartBuiltIn = "transmart-builtin"
	ICibaBuiltIn     = "iciba-builtin"
)

type Field struct {
	Key, Label, Default string
	Secret, Required    bool
}

type Provider struct {
	ID, Name string
	AI       bool
	BuiltIn  bool
	Fields   []Field
}

var Providers = []Provider{
	{ID: MicrosoftBuiltIn, Name: "微软翻译（内置）", BuiltIn: true},
	{ID: GoogleBuiltIn, Name: "谷歌翻译（内置）", BuiltIn: true},
	{ID: YandexBuiltIn, Name: "Yandex 翻译（内置）", BuiltIn: true},
	{ID: TransmartBuiltIn, Name: "腾讯翻译（内置）", BuiltIn: true},
	{ID: ICibaBuiltIn, Name: "金山翻译（内置）", BuiltIn: true},
	{ID: DeepL, Name: "DeepL", Fields: []Field{apiKey(true)}},
	{ID: Baidu, Name: "百度翻译", Fields: appCredentials()},
	{ID: AI, Name: "OpenAI 兼容", AI: true, Fields: modelFields("https://api.openai.com/v1", false)},
	{ID: Azure, Name: "微软 Azure Translator", Fields: []Field{apiKey(true), {Key: "region", Label: "资源区域（按需填写）"}}},
	{ID: Google, Name: "谷歌 Google Cloud Translation", Fields: []Field{apiKey(true)}},
	{ID: Yandex, Name: "Yandex Translate", Fields: []Field{
		{Key: "apiKey", Label: "API Key 或 IAM Token", Secret: true, Required: true},
		{Key: "folderID", Label: "文件夹 ID（使用 IAM Token 时填写）"},
	}},
	{ID: Youdao, Name: "有道翻译", Fields: appCredentials()},
	{ID: Hunyuan, Name: "腾讯混元翻译", Fields: []Field{apiKey(true), {Key: "model", Label: "模型 ID", Default: "hy-mt2-plus", Required: true}}},
	{ID: MTran, Name: "MTranServer", Fields: []Field{
		{Key: "baseURL", Label: "服务地址", Default: "http://localhost:8989", Required: true},
		{Key: "apiKey", Label: "访问令牌（按需填写）", Secret: true},
	}},
	{ID: QwenMT, Name: "Qwen-MT", Fields: modelFields("https://dashscope.aliyuncs.com/compatible-mode/v1", true)},
	{ID: Zhipu, Name: "智谱", AI: true, Fields: modelFields("https://open.bigmodel.cn/api/paas/v4", true)},
	{ID: DeepSeek, Name: "DeepSeek", AI: true, Fields: modelFields("https://api.deepseek.com", true)},
}

func ProviderByID(id string) (Provider, bool) {
	for _, p := range Providers {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

func apiKey(required bool) Field {
	return Field{Key: "apiKey", Label: "API Key", Secret: true, Required: required}
}

func appCredentials() []Field {
	return []Field{
		{Key: "appID", Label: "应用 ID", Required: true},
		{Key: "appSecret", Label: "应用密钥", Secret: true, Required: true},
	}
}

func modelFields(endpoint string, keyRequired bool) []Field {
	return []Field{
		{Key: "baseURL", Label: "服务地址", Default: endpoint, Required: true},
		{Key: "model", Label: "模型 ID", Required: true},
		apiKey(keyRequired),
	}
}
