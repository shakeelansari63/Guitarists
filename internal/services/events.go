package services

import "github.com/wailsapp/wails/v3/pkg/application"

// emit sends an event to the frontend. It is safe to call before the
// application has finished starting (the event is simply dropped).
func emit(name string, data any) {
	app := application.Get()
	if app == nil || app.Event == nil {
		return
	}
	app.Event.Emit(name, data)
}
