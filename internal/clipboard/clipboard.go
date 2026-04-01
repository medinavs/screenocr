package clipboard

import (
	"context"
	"fmt"
	"log"

	cb "golang.design/x/clipboard"
)

// Init initializes the clipboard system. Must be called once at startup.
func Init() error {
	err := cb.Init()
	if err != nil {
		return fmt.Errorf("clipboard init failed: %w", err)
	}
	log.Println("[clipboard] initialized")
	return nil
}

// CopyText copies the given text to the system clipboard.
func CopyText(text string) error {
	if len(text) == 0 {
		return fmt.Errorf("empty text, nothing to copy")
	}

	cb.Write(cb.FmtText, []byte(text))
	log.Printf("[clipboard] copied %d characters", len(text))
	return nil
}

// ReadText reads text from the system clipboard.
func ReadText() (string, error) {
	data := cb.Read(cb.FmtText)
	if data == nil {
		return "", fmt.Errorf("clipboard is empty or not text")
	}
	return string(data), nil
}

// WatchText watches the clipboard for text changes.
func WatchText(ctx context.Context) <-chan string {
	ch := make(chan string)
	raw := cb.Watch(ctx, cb.FmtText)

	go func() {
		defer close(ch)
		for data := range raw {
			ch <- string(data)
		}
	}()

	return ch
}
