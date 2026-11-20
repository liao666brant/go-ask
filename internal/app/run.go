package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"goask/internal/data"
)

// Run opens the user profile and runs the background desktop application.
func Run(trayIcon []byte) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "go-ask")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	a := &application{configPath: filepath.Join(dir, "settings.json"), pref: preferences{Hotkey: "ctrl+q", DeepL: true, Baidu: true, AI: true, BaseURL: "https://api.openai.com/v1"}}
	raw, err := os.ReadFile(a.configPath)
	if err == nil {
		if err = json.Unmarshal(raw, &a.pref); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	a.db, err = data.Open(filepath.Join(dir, "history.db"))
	if err != nil {
		return err
	}
	defer a.db.Close()
	runDesktop(a, trayIcon)
	return nil
}
