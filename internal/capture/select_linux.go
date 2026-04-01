//go:build linux

package capture

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

// selectRegionLinux uses `slop` (Select Operation) for region selection.
// slop is a lightweight tool that provides a click-and-drag selection UI.
// Install: sudo apt install slop
// Fallback: xdotool-based manual tracking.
func selectRegion() (Region, error) {
	log.Println("[capture] starting Linux region selection")

	if path, err := exec.LookPath("slop"); err == nil {
		log.Printf("[capture] using slop at %s", path)
		return selectWithSlop()
	}

	// fallback: xdotool-based selection
	log.Println("[capture] slop not found, falling back to xdotool")
	return selectWithXdotool()
}

func selectWithSlop() (Region, error) {
	// slop outputs: X Y W H in format "XxY+W+H" or configurable
	cmd := exec.Command("slop", "-f", "%x %y %w %h", "--highlight", "--tolerance=0")
	out, err := cmd.Output()
	if err != nil {
		return Region{}, fmt.Errorf("slop selection failed: %w", err)
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 4 {
		return Region{}, fmt.Errorf("unexpected slop output: %s", string(out))
	}

	x, _ := strconv.Atoi(parts[0])
	y, _ := strconv.Atoi(parts[1])
	w, _ := strconv.Atoi(parts[2])
	h, _ := strconv.Atoi(parts[3])

	if w < 5 || h < 5 {
		return Region{}, fmt.Errorf("selected region too small: %dx%d", w, h)
	}

	log.Printf("[capture] slop region: (%d,%d) %dx%d", x, y, w, h)
	return Region{X: x, Y: y, W: w, H: h}, nil
}

func selectWithXdotool() (Region, error) {
	fmt.Println("Click and drag to select a region...")

	cmd := exec.Command("xdotool", "getmouselocation", "--shell")
	out, err := cmd.Output()
	if err != nil {
		return Region{}, fmt.Errorf("xdotool failed: %w", err)
	}

	coords := parseXdotoolOutput(string(out))

	return Region{
		X: coords["X"],
		Y: coords["Y"],
		W: 400,
		H: 300,
	}, nil
}

func parseXdotoolOutput(output string) map[string]int {
	result := make(map[string]int)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			val, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			result[strings.TrimSpace(parts[0])] = val
		}
	}
	return result
}
