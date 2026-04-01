package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const defaultOllamaURL = "http://localhost:11434"

type OllamaEngine struct {
	model   string
	baseURL string
	client  *http.Client
}

// model: e.g. "moondream", "minicpm-v", "llava"
func NewOllamaEngine(model, baseURL string) *OllamaEngine {
	if baseURL == "" {
		baseURL = defaultOllamaURL
	}
	return &OllamaEngine{
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (e *OllamaEngine) Available() bool {
	resp, err := e.client.Get(e.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return strings.Contains(string(body), e.model)
}

type ollamaRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
	Stream bool     `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

const codePrompt = `You are an OCR engine specialized in extracting code from screenshots.
Extract ALL text visible in this image exactly as it appears.
Preserve indentation, spacing, punctuation, and every character precisely.
Do not summarize, explain, or add anything — output only the raw extracted text.`

func (e *OllamaEngine) ProcessImage(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}
	return e.processBytes(buf.Bytes())
}

func (e *OllamaEngine) processBytes(data []byte) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(data)

	body, err := json.Marshal(ollamaRequest{
		Model:  e.model,
		Prompt: codePrompt,
		Images: []string{b64},
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	log.Printf("[ocr/ollama] sending to model %q (%d KB)", e.model, len(data)/1024)

	resp, err := e.client.Post(
		e.baseURL+"/api/generate",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("ollama error: %s", result.Error)
	}

	text := strings.TrimSpace(result.Response)
	log.Printf("[ocr/ollama] extracted %d characters", len(text))
	return text, nil
}
