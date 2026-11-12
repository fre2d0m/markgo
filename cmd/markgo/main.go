package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	markgoapp "github.com/xddcode/markgo/internal/app"
)

func main() {
	a := markgoapp.New()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:           "MarkGo",
			Width:           1320,
			Height:          840,
			MinWidth:        960,
			MinHeight:       640,
			StateKey:        "main",
			BackgroundColor: "light-dark(#fafafc, #18181b)",
			Content:         ui.View(a.View),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
