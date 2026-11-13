package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	markgoapp "github.com/xddcode/markgo/internal/app"
)

func main() {
	a := markgoapp.New()
	if os.Getenv("MARKGO_EPHEMERAL") != "1" {
		if err := a.LoadAppearance(); err != nil {
			log.Printf("Could not restore appearance: %v", err)
		}
	}
	mygo.Theme.SetSource(a.ThemeSource())
	mygo.App.OnOpenFile(a.OpenFile)
	mygo.App.WhenReady(func() {
		stateKey := "main"
		if os.Getenv("MARKGO_EPHEMERAL") == "1" {
			stateKey = ""
		}
		window := mygo.NewWindow(a.WindowOptions(stateKey))
		a.Attach(window)
	})
	err := mygo.App.Run()
	a.WaitForPreferences()
	if err != nil {
		log.Fatal(err)
	}
}
