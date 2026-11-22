package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/core"
)

type preferences struct {
	Hotkey           string
	PanelIdleSeconds int
	HidePanelOnBlur  bool
	Providers        map[string]providerPreference
	ChatProvider     string `json:",omitempty"`
	// Legacy service fields are read until a service is saved in the new format.
	DeepL                bool
	Baidu                bool
	AI                   bool
	BaseURL              string
	Model                string
	BaiduAppID           string
	AlternateCredentials bool `json:",omitempty"`
}

type providerPreference struct {
	Fields         map[string]string `json:",omitempty"`
	Translate      bool
	CredentialSlot int `json:",omitempty"`
}

type credentials struct {
	Providers   map[string]map[string]string `json:"-"`
	DeepLKey    string                       `json:",omitempty"`
	BaiduSecret string                       `json:",omitempty"`
	APIKey      string                       `json:",omitempty"`
}

func loadPreferences(path string) (preferences, error) {
	pref := preferences{
		Hotkey: "ctrl+q", PanelIdleSeconds: 30, HidePanelOnBlur: true,
		DeepL: true, Baidu: true, AI: true, BaseURL: "https://api.openai.com/v1",
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		pref.Providers = make(map[string]providerPreference)
		return pref, nil
	}
	if err != nil {
		return preferences{}, err
	}
	err = json.Unmarshal(raw, &pref)
	return pref, err
}

// Older settings have no slot field and use the original keychain account.
func (p preferences) credentialAccount() string {
	if p.AlternateCredentials {
		return "providers-alternate"
	}
	return "providers"
}

func providerCredentialAccount(id string, slot int) string {
	switch slot {
	case 1:
		return "providers/" + id
	case 2:
		return "providers-alternate/" + id
	default:
		return ""
	}
}

func (a *application) chatProviderID() string {
	if a.pref.Providers == nil {
		return service.AI
	}
	return a.pref.ChatProvider
}

func (a *application) hasProvider(id string) bool {
	if a.pref.Providers != nil {
		_, ok := a.pref.Providers[id]
		return ok
	}
	switch id {
	case service.DeepL:
		return a.keys.DeepLKey != ""
	case service.Baidu:
		return a.pref.BaiduAppID != "" || a.keys.BaiduSecret != ""
	case service.AI:
		return a.pref.Model != "" || a.keys.APIKey != "" ||
			(a.pref.BaseURL != "" && a.pref.BaseURL != "https://api.openai.com/v1")
	default:
		return false
	}
}

func (a *application) providerSettings(id string) (providerPreference, map[string]string) {
	pref := providerPreference{Fields: make(map[string]string)}
	keys := make(map[string]string)
	p, _ := service.ProviderByID(id)
	for _, field := range p.Fields {
		if field.Default != "" {
			pref.Fields[field.Key] = field.Default
		}
	}
	if a.pref.Providers != nil {
		previous := a.pref.Providers[id]
		pref.Translate = previous.Translate
		maps.Copy(pref.Fields, previous.Fields)
		maps.Copy(keys, a.keys.Providers[id])
		return pref, keys
	}
	switch id {
	case service.DeepL:
		pref.Translate = a.pref.DeepL
		keys["apiKey"] = a.keys.DeepLKey
	case service.Baidu:
		pref.Translate = a.pref.Baidu
		pref.Fields["appID"] = a.pref.BaiduAppID
		keys["appSecret"] = a.keys.BaiduSecret
	case service.AI:
		pref.Translate = a.pref.AI
		pref.Fields["baseURL"], pref.Fields["model"] = a.pref.BaseURL, a.pref.Model
		keys["apiKey"] = a.keys.APIKey
	}
	return pref, keys
}

func validateProvider(p service.Provider, fields, keys map[string]string) error {
	for _, field := range p.Fields {
		value := fields[field.Key]
		if field.Secret {
			value = keys[field.Key]
		}
		if field.Required && strings.TrimSpace(value) == "" {
			return fmt.Errorf("请填写%s", field.Label)
		}
	}
	if address := fields["baseURL"]; address != "" {
		endpoint, err := service.ChatEndpoint(address)
		if err != nil {
			return errors.New("请填写有效的 HTTP 或 HTTPS 服务地址")
		}
		if p.ID == service.AI && strings.TrimSpace(keys["apiKey"]) == "" {
			u, _ := url.Parse(endpoint)
			if strings.EqualFold(strings.TrimSuffix(u.Hostname(), "."), "api.openai.com") {
				return errors.New("请填写 OpenAI API Key")
			}
		}
	}
	return nil
}

func (a *application) providerConfig(id string) (service.Config, error) {
	p, ok := service.ProviderByID(id)
	if !ok {
		return service.Config{}, errors.New("请选择服务提供商")
	}
	pref, keys := a.providerSettings(id)
	if a.pref.Providers[id].CredentialSlot != 0 && len(keys) == 0 {
		return service.Config{}, errors.New("此服务的凭据不可用")
	}
	f := pref.Fields
	if err := validateProvider(p, f, keys); err != nil {
		return service.Config{}, err
	}
	return service.Config{
		BaseURL: f["baseURL"], Model: f["model"], APIKey: keys["apiKey"],
		Region: f["region"], FolderID: f["folderID"], AppID: f["appID"], AppSecret: keys["appSecret"],
		DeepLKey: keys["apiKey"], BaiduAppID: f["appID"], BaiduSecret: keys["appSecret"],
	}, nil
}

func (a *application) savePreferences(cfg preferences) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(a.configPath), ".settings-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(raw)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(f.Name(), a.configPath); err != nil {
		return err
	}
	a.pref = cfg
	return nil
}

func (a *application) saveProvider(store core.Secrets, id string, draft providerPreference, secrets map[string]string, makeChat bool) error {
	p, ok := service.ProviderByID(id)
	if !ok || makeChat && !p.AI {
		return errors.New("请选择可用的服务提供商")
	}
	fields := make(map[string]string)
	private := make(map[string]string)
	for _, f := range p.Fields {
		if f.Secret {
			if value := strings.TrimSpace(secrets[f.Key]); value != "" {
				private[f.Key] = value
			}
		} else {
			fields[f.Key] = strings.TrimSpace(draft.Fields[f.Key])
		}
	}
	if err := validateProvider(p, fields, private); err != nil {
		return err
	}
	migrating := a.pref.Providers == nil
	if migrating && !a.credentialsLoaded {
		return errors.New(a.keyStoreError + "。请解锁系统凭据存储并重新打开设置。")
	}
	if a.pref.Providers[id].CredentialSlot != 0 && len(a.keys.Providers[id]) == 0 && len(private) == 0 {
		return errors.New("无法读取此服务的已有凭据，请解锁凭据存储或重新填写密钥")
	}
	next := a.pref
	next.Providers = make(map[string]providerPreference)
	maps.Copy(next.Providers, a.pref.Providers)
	keys := credentials{Providers: make(map[string]map[string]string)}
	maps.Copy(keys.Providers, a.keys.Providers)
	var obsolete []string
	if migrating {
		for _, legacy := range []string{service.DeepL, service.Baidu, service.AI} {
			if a.hasProvider(legacy) {
				entry, entryKeys := a.providerSettings(legacy)
				maps.DeleteFunc(entryKeys, func(_, value string) bool { return value == "" })
				next.Providers[legacy], keys.Providers[legacy] = entry, entryKeys
			}
		}
		if _, exists := next.Providers[service.AI]; exists {
			next.ChatProvider = service.AI
		}
		next.DeepL, next.Baidu, next.AI = false, false, false
		next.BaseURL, next.Model, next.BaiduAppID = "", "", ""
		obsolete = append(obsolete, a.pref.credentialAccount())
		next.AlternateCredentials = false
	}
	current := next.Providers[id]
	current.Fields, current.Translate = fields, draft.Translate
	next.Providers[id] = current
	if makeChat {
		next.ChatProvider = id
	}
	if !migrating && maps.Equal(private, keys.Providers[id]) {
		return a.savePreferences(next)
	}
	keys.Providers[id] = private

	// Each provider uses its own credential records so their combined size is
	// independent of the platform's limit for one record.
	var staged []string
	for _, provider := range service.Providers {
		if !migrating && provider.ID != id {
			continue
		}
		entry, exists := next.Providers[provider.ID]
		if !exists {
			continue
		}
		if account := providerCredentialAccount(provider.ID, entry.CredentialSlot); account != "" {
			obsolete = append(obsolete, account)
		}
		entryKeys := keys.Providers[provider.ID]
		if len(entryKeys) == 0 {
			entry.CredentialSlot = 0
			next.Providers[provider.ID] = entry
			continue
		}
		if entry.CredentialSlot == 1 {
			entry.CredentialSlot = 2
		} else {
			entry.CredentialSlot = 1
		}
		account := providerCredentialAccount(provider.ID, entry.CredentialSlot)
		raw, err := json.Marshal(entryKeys)
		if err == nil {
			err = store.Set(account, string(raw))
		}
		if err != nil {
			return errors.Join(fmt.Errorf("%s密钥保存失败：%w", provider.Name, err), deleteCredentialAccounts(store, staged))
		}
		staged = append(staged, account)
		next.Providers[provider.ID] = entry
	}
	if err := a.savePreferences(next); err != nil {
		return errors.Join(err, deleteCredentialAccounts(store, staged))
	}
	a.keys = keys
	a.keyStoreError = ""
	if err := deleteCredentialAccounts(store, obsolete); err != nil {
		return fmt.Errorf("服务设置已保存，旧凭据未能清理：%w", err)
	}
	return nil
}

func deleteCredentialAccounts(store core.Secrets, accounts []string) error {
	var errorsFound []error
	for _, account := range accounts {
		if err := store.Delete(account); err != nil {
			errorsFound = append(errorsFound, err)
		}
	}
	return errors.Join(errorsFound...)
}

func (a *application) loadCredentials(cx *core.Scope) {
	if a.credentialsLoaded {
		return
	}
	s := cx.Secrets()
	if a.pref.Providers != nil {
		a.keyStoreError = ""
		a.keys.Providers = make(map[string]map[string]string)
		for _, provider := range service.Providers {
			slot := a.pref.Providers[provider.ID].CredentialSlot
			if slot == 0 {
				continue
			}
			account := providerCredentialAccount(provider.ID, slot)
			var keys map[string]string
			if account != "" {
				if raw, ok, err := s.Get(account); err == nil && ok {
					if json.Unmarshal([]byte(raw), &keys) != nil {
						keys = nil
					}
				}
			}
			if len(keys) == 0 {
				a.keyStoreError = "无法读取" + provider.Name + "的凭据"
				continue
			}
			a.keys.Providers[provider.ID] = keys
		}
		a.credentialsLoaded = a.keyStoreError == ""
		return
	}
	if raw, ok, err := s.Get(a.pref.credentialAccount()); err != nil {
		a.keyStoreError = "凭据存储不可用：" + err.Error()
		return
	} else if ok {
		var keys credentials
		if err = json.Unmarshal([]byte(raw), &keys); err != nil {
			a.keyStoreError = "无法读取服务凭据：" + err.Error()
			return
		}
		a.keys = keys
	}
	a.credentialsLoaded = true
	a.keyStoreError = ""
}
