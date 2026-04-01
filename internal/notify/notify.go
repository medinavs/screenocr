package notify

import (
	"log"

	"github.com/gen2brain/beeep"
)

const appName = "ScreenOCR"

func Success(message string) {
	err := beeep.Notify(appName, message, "")
	if err != nil {
		log.Printf("[notify] failed to show notification: %v", err)
	}
}

func Error(message string) {
	err := beeep.Alert(appName, message, "")
	if err != nil {
		log.Printf("[notify] failed to show error notification: %v", err)
	}
}

func Info(message string) {
	err := beeep.Notify(appName, message, "")
	if err != nil {
		log.Printf("[notify] failed to show info notification: %v", err)
	}
}
