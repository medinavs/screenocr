package hotkey

import (
	"context"
	"fmt"
	"log"

	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

type Modifier = hotkey.Modifier
type Key = hotkey.Key

type Callback func()

type Handler struct {
	hk       *hotkey.Hotkey
	callback Callback
	cancel   context.CancelFunc
}

func New(mods []Modifier, key Key, cb Callback) *Handler {
	return &Handler{
		callback: cb,
		hk:       hotkey.New(mods, key),
	}
}

// Start registers the hotkey and begins listening in a goroutine.
// Must be called from the main thread on macOS (use mainthread.Init).
func (h *Handler) Start(ctx context.Context) error {
	err := h.hk.Register()
	if err != nil {
		return fmt.Errorf("hotkey register failed: %w", err)
	}

	ctx, h.cancel = context.WithCancel(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.hk.Keydown():
				log.Println("[hotkey] triggered")
				h.callback()
			}
		}
	}()

	log.Printf("[hotkey] registered and listening")
	return nil
}

// Stop unregisters the hotkey.
func (h *Handler) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	if h.hk != nil {
		h.hk.Unregister()
	}
	log.Println("[hotkey] unregistered")
}

// RunOnMainThread wraps mainthread.Init for macOS compatibility.
// fn is the app's real main logic; it runs on a background goroutine
// while the main thread is reserved for system event handling.
func RunOnMainThread(fn func()) {
	mainthread.Init(fn)
}
