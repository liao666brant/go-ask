package app

import (
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
	path := filepath.Join(dir, "settings.json")
	pref, err := loadPreferences(path)
	if err != nil {
		return err
	}
	a := &application{configPath: path, pref: pref}
	a.db, err = data.Open(filepath.Join(dir, "history.db"))
	if err != nil {
		return err
	}
	defer a.db.Close()
	runDesktop(a, trayIcon)
	return nil
}
