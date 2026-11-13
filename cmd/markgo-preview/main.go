// MarkGo Writing Preview opens only in-memory samples. It is a separate
// executable so visual/native acceptance cannot accidentally open real files.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/xddcode/markgo/internal/app"
)

func main() {
	preview := app.NewTypesettingPreview()
	mygo.Theme.SetSource(mygo.ThemeLight)
	mygo.App.WhenReady(func() {
		window := mygo.NewWindow(preview.WindowOptions())
		if err := app.SetWindowIcon(window); err != nil {
			log.Printf("Could not set window icon: %v", err)
		}
		preview.Attach(window)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
