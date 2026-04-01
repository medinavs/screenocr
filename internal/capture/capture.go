package capture

import (
	"fmt"
	"image"
	"log"

	"github.com/kbinani/screenshot"
)

type Region struct {
	X, Y, W, H int
}

// CaptureFullScreen captures the entire primary display.
func CaptureFullScreen() (*image.RGBA, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("screen capture failed: %w", err)
	}

	log.Printf("[capture] full screen captured: %dx%d", bounds.Dx(), bounds.Dy())
	return img, nil
}

// CaptureRegion captures a specific rectangular region from the screen.
// Coordinates are in screen pixels.
func CaptureRegion(r Region) (*image.RGBA, error) {
	if r.W <= 0 || r.H <= 0 {
		return nil, fmt.Errorf("invalid region: width=%d height=%d", r.W, r.H)
	}

	rect := image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, fmt.Errorf("region capture failed: %w", err)
	}

	log.Printf("[capture] region captured: %v", rect)
	return img, nil
}

// GetDisplayBounds returns the primary display's bounds.
func GetDisplayBounds() image.Rectangle {
	if screenshot.NumActiveDisplays() == 0 {
		return image.Rectangle{}
	}
	return screenshot.GetDisplayBounds(0)
}

// SelectRegion opens an overlay for the user to draw a rectangle.
// This is platform-specific. On Windows, it uses a transparent fullscreen window.
// Returns the selected region in screen coordinates.
func SelectRegion() (Region, error) {
	return selectRegion()
}
