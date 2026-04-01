//go:build !windows

package tray

import "log"

func Run(onReady func(), onQuit func()) {
	log.Println("[tray] system tray not supported on this platform without CGO")
	onReady()
}

// Setup is a no-op on non-Windows platforms.
func Setup() <-chan struct{} {
	return make(chan struct{})
}
