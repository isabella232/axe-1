package throwing

import (
	"github.com/gdamore/tcell"
)

var (
	EscapeEventHandler = func(app *AppView) func(event *tcell.EventKey) *tcell.EventKey {
		return func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEscape {
				app.showMenu = false
				app.SwitchPage(app.currentPage, app.CurrentPage())
			}
			return event
		}
	}

	searchDoneEventHandler = func(app *AppView, t *TableView) func(key tcell.Key) {
		return func(key tcell.Key) {
			switch key {
			case tcell.KeyEscape:
				app.SetFocus(app.content)
				app.searchView.InputField.SetText("")
			case tcell.KeyEnter:
				t.UpdateWithSearch(app.searchView.InputField.GetText())
				app.searchView.InputField.SetText("")
				t.Refresh()
			}
		}
	}
)