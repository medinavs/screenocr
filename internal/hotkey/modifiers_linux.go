//go:build linux && cgo

package hotkey

import "golang.design/x/hotkey"

func DefaultModifiers() []Modifier {
	return []Modifier{hotkey.ModCtrl, hotkey.ModShift}
}

func DefaultKey() Key {
	return hotkey.KeyO
}
