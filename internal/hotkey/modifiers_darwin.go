//go:build darwin && cgo

package hotkey

import "golang.design/x/hotkey"

func DefaultModifiers() []Modifier {
	return []Modifier{hotkey.ModCmd, hotkey.ModShift}
}

func DefaultKey() Key {
	return hotkey.KeyO
}
