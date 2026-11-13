package app

import (
	"github.com/egoist/mygo"
	markgoresources "github.com/xddcode/markgo/resources"
)

func SetWindowIcon(window *mygo.Window) error {
	return window.SetIcon(markgoresources.AppIconPNG)
}
