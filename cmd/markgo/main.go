package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	markgoapp "github.com/xddcode/markgo/internal/app"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--typesetting-preview" {
		preview := markgoapp.NewTypesettingPreview()
		mygo.Theme.SetSource(mygo.ThemeLight)
		mygo.App.WhenReady(func() { preview.Attach(mygo.NewWindow(preview.WindowOptions())) })
		if err := mygo.App.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}
	ephemeral := os.Getenv("MARKGO_EPHEMERAL") == "1"
	if !ephemeral && !mygo.App.RequestSingleInstanceLock() {
		return
	}
	a := markgoapp.New()
	if !ephemeral {
		if err := a.LoadAppearance(); err != nil {
			log.Printf("Could not restore appearance: %v", err)
		}
		if err := a.LoadSession(); err != nil {
			log.Printf("Could not restore session: %v", err)
		}
	}
	mygo.Theme.SetSource(a.ThemeSource())
	mygo.App.OnOpenFile(a.OpenFile)
	mygo.App.WhenReady(func() {
		stateKey := "main"
		if ephemeral {
			stateKey = ""
		}
		window := mygo.NewWindow(a.WindowOptions(stateKey))
		a.Attach(window)
		mygo.App.OnSecondInstance(func(_ []string, _ string) {
			window.Restore()
			window.Focus()
		})
	})
	err := mygo.App.Run()
	a.WaitForPreferences()
	a.WaitForSession()
	if err != nil {
		log.Fatal(err)
	}
}
