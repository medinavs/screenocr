//go:build !windows && !cgo

package hotkey

func DefaultModifiers() []Modifier {
	return []Modifier{}
}

func DefaultKey() Key {
	return 0
}
