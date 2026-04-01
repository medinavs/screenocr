//go:build windows

package tray

import (
	_ "embed"
	"log"

	"github.com/getlantern/systray"
)

//go:embed icon.ico
var trayIcon []byte

func Run(onReady func(), onQuit func()) {
	systray.Run(onReady, onQuit)
}

func Setup() <-chan struct{} {
	quitCh := make(chan struct{})

	systray.SetIcon(trayIcon)
	systray.SetTooltip("ScreenOCR — Ctrl+Shift+O to capture")

	mStatus := systray.AddMenuItem("ScreenOCR Running", "")
	mStatus.Disable()

	systray.AddSeparator()

	mHotkey := systray.AddMenuItem("Hotkey: Ctrl+Shift+O", "")
	mHotkey.Disable()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Exit ScreenOCR")

	go func() {
		<-mQuit.ClickedCh
		log.Println("[tray] quit clicked")
		close(quitCh)
		systray.Quit()
	}()

	log.Println("[tray] system tray initialized")
	return quitCh
}
