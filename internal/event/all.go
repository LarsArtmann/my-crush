package event

import (
	"time"
)

var appStartTime time.Time

func AppInitialized() {
	appStartTime = time.Now()
	send("app initialized")
}

// AppStartTime returns the time when the app was initialized.
// This is set by event.AppInitialized() and should be used
// for any uptime tracking that needs to start from the actual
// app start time rather than component creation time.
func AppStartTime() time.Time {
	return appStartTime
}

func AppExited() {
	duration := time.Since(appStartTime).Truncate(time.Second)
	send(
		"app exited",
		"app duration pretty", duration.String(),
		"app duration in seconds", int64(duration.Seconds()),
	)
	Flush()
}

func SessionCreated() {
	send("session created")
}

func SessionDeleted() {
	send("session deleted")
}

func SessionSwitched() {
	send("session switched")
}

func FilePickerOpened() {
	send("filepicker opened")
}

func PromptSent(props ...any) {
	send(
		"prompt sent",
		props...,
	)
}

func PromptResponded(props ...any) {
	send(
		"prompt responded",
		props...,
	)
}

func TokensUsed(props ...any) {
	send(
		"tokens used",
		props...,
	)
}
