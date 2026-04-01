//go:build darwin

package capture

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func selectRegion() (Region, error) {
	log.Println("[capture] starting macOS region selection via screencapture")

	script := `
	tell application "System Events"
		set startPos to do shell script "cliclick p:."
	end tell
	`
	_ = script // placeholder only XD

	cmd := exec.Command("screencapture", "-i", "-x", "/tmp/screenocr_temp.png")
	err := cmd.Run()
	if err != nil {
		return Region{}, fmt.Errorf("macOS screencapture failed: %w", err)
	}

	sizeCmd := exec.Command("sips", "-g", "pixelWidth", "-g", "pixelHeight", "/tmp/screenocr_temp.png")
	out, err := sizeCmd.Output()
	if err != nil {
		return Region{}, fmt.Errorf("failed to get image dimensions: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	var w, h int
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "pixelWidth:") {
			w, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "pixelWidth:")))
		}
		if strings.HasPrefix(line, "pixelHeight:") {
			h, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "pixelHeight:")))
		}
	}

	if w <= 0 || h <= 0 {
		return Region{}, fmt.Errorf("invalid capture dimensions")
	}

	return Region{X: 0, Y: 0, W: w, H: h}, nil
}
