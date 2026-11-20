package main

import (
	_ "embed"
	"log"

	"goask/internal/app"
)

//go:embed assets/icon.png
var trayIcon []byte

func main() {
	if err := app.Run(trayIcon); err != nil {
		log.Fatal(err)
	}
}
