package ocr

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	Language string

	// PageSegMode controls Tesseract's page segmentation mode.
	// PSM 6 = Assume a single uniform block of text (best for code).
	// PSM 4 = Assume a single column of text of variable sizes.
	// PSM 3 = Fully automatic (default Tesseract behavior).
	PageSegMode int

	// Whitelist limits recognized characters (empty = all).
	// For code: include all printable ASCII.
	Whitelist string

	// PreserveInterwordSpaces keeps spacing intact (important for code).
	PreserveInterwordSpaces bool
}

func DefaultConfig() Config {
	return Config{
		Language:                "eng",
		PageSegMode:             6, // Uniform text block
		PreserveInterwordSpaces: true,
		Whitelist:               "", // Allow all characters for code
	}
}

type Engine struct {
	config Config
}

func NewEngine(cfg Config) *Engine {
	return &Engine{config: cfg}
}

// ProcessImage runs OCR on an in-memory image and returns the extracted text.
func (e *Engine) ProcessImage(img image.Image) (string, error) {
	log.Println("[ocr] encoding image to PNG in memory")

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	log.Printf("[ocr] PNG size: %d bytes", buf.Len())
	return e.ProcessImageBytes(buf.Bytes())
}

// ProcessImageBytes runs OCR on raw PNG/image bytes.
func (e *Engine) ProcessImageBytes(data []byte) (string, error) {
	// write to a temp file — tesseract CLI requires a file path input
	tmp, err := os.CreateTemp("", "screenocr-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmp.Close()

	args := []string{
		tmp.Name(),
		"stdout",
		"-l", e.config.Language,
		"--psm", fmt.Sprintf("%d", e.config.PageSegMode),
	}

	if e.config.PreserveInterwordSpaces {
		args = append(args, "--oem", "3", "-c", "preserve_interword_spaces=1")
	}

	if e.config.Whitelist != "" {
		args = append(args, "-c", "tessedit_char_whitelist="+e.config.Whitelist)
	}

	out, err := exec.Command("tesseract", args...).Output()
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w", err)
	}

	text := strings.TrimSpace(string(out))
	log.Printf("[ocr] extracted %d characters", len(text))
	return text, nil
}
