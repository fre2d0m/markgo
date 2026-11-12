package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	markgoapp "github.com/xddcode/markgo/internal/app"
)

func main() {
	a := markgoapp.New()
	mygo.App.OnOpenFile(a.OpenFile)
	mygo.App.WhenReady(func() {
		stateKey := "main"
		if os.Getenv("MARKGO_EPHEMERAL") == "1" {
			stateKey = ""
		}
		window := mygo.NewWindow(mygo.WindowOptions{
			Title:           "MarkGo",
			Width:           1320,
			Height:          840,
			MinWidth:        960,
			MinHeight:       640,
			StateKey:        stateKey,
			BackgroundColor: "light-dark(#fafafc, #18181b)",
			Content:         ui.View(a.View),
		})
		a.Attach(window)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
