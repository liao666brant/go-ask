package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"goask/internal/service"

	"github.com/nsgo-dev/nsgo/core"
)

type preferences struct {
	Hotkey                     string
	DeepL, Baidu, AI           bool
	BaseURL, Model, BaiduAppID string
	AlternateCredentials       bool `json:",omitempty"`
}

type credentials struct{ DeepLKey, BaiduSecret, APIKey string }

// Older settings have no slot field and use the original keychain account.
func (p preferences) credentialAccount() string {
	if p.AlternateCredentials {
		return "providers-alternate"
	}
	return "providers"
}

func (a *application) serviceConfig() service.Config {
	return service.Config{DeepLKey: a.keys.DeepLKey, BaiduAppID: a.pref.BaiduAppID, BaiduSecret: a.keys.BaiduSecret, BaseURL: a.pref.BaseURL, APIKey: a.keys.APIKey, Model: a.pref.Model}
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

func (a *application) saveServiceSettings(store core.Secrets, next preferences, keys credentials) error {
	next.Hotkey = a.pref.Hotkey
	next.AlternateCredentials = a.pref.AlternateCredentials
	if keys == a.keys {
		return a.savePreferences(next)
	}
	if !a.credentialsLoaded {
		return errors.New(a.keyStoreError + "。请解锁系统凭据存储并重新打开设置。")
	}

	// Stage the new keys before switching the endpoint and its account together.
	oldAccount := a.pref.credentialAccount()
	next.AlternateCredentials = !next.AlternateCredentials
	newAccount := next.credentialAccount()
	raw, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	if err := store.Set(newAccount, string(raw)); err != nil {
		return fmt.Errorf("服务密钥保存失败：%w", err)
	}
	if err := a.savePreferences(next); err != nil {
		if cleanupErr := store.Delete(newAccount); cleanupErr != nil {
			return errors.Join(err, fmt.Errorf("未启用的凭据未能清理：%w", cleanupErr))
		}
		return err
	}
	a.keys = keys
	a.keyStoreError = ""
	if err := store.Delete(oldAccount); err != nil {
		return fmt.Errorf("服务设置已保存，旧凭据未能清理：%w", err)
	}
	return nil
}

func (a *application) loadCredentials(cx *core.Scope) {
	if a.credentialsLoaded {
		return
	}
	s := cx.Secrets()
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
