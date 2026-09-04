package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.NewWithID("io.github.wadhah.weztermconfigurator")
	w := a.NewWindow("WezTerm Configurator")
	w.Resize(fyne.NewSize(1280, 820))
	w.ShowAndRun()
}
