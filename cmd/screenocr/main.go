package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/medinavs/screenocr/internal/capture"
	"github.com/medinavs/screenocr/internal/cleanup"
	"github.com/medinavs/screenocr/internal/clipboard"
	hk "github.com/medinavs/screenocr/internal/hotkey"
	"github.com/medinavs/screenocr/internal/notify"
	"github.com/medinavs/screenocr/internal/ocr"
	"github.com/medinavs/screenocr/internal/tray"
)

var (
	flagCleanup     = flag.Bool("cleanup", true, "Apply code OCR cleanup heuristics")
	flagLineNumbers = flag.Bool("strip-linenum", true, "Strip detected line numbers")
	flagPSM         = flag.Int("psm", 6, "Tesseract page segmentation mode (6=uniform block)")
	flagLang        = flag.String("lang", "eng", "Tesseract language")
	flagNoTray      = flag.Bool("no-tray", false, "Run without system tray icon")
	flagVerbose     = flag.Bool("verbose", false, "Enable verbose logging")
	flagOllama      = flag.Bool("ollama", true, "Use Ollama vision model when available (falls back to Tesseract)")
	flagOllamaModel = flag.String("ollama-model", "minicpm-v", "Ollama model to use (moondream, minicpm-v, llava)")
	flagOllamaURL   = flag.String("ollama-url", "http://localhost:11434", "Ollama API base URL")
	flagDebug       = flag.Bool("debug", false, "Save raw + preprocessed images to %TEMP%\\screenocr_debug\\ for inspection")
	flagForceInvert = flag.Bool("force-invert", false, "Always invert image before OCR (use on dark VDI themes)")
	flagNoInvert    = flag.Bool("no-invert", false, "Never invert image before OCR")
)

type app struct {
	tesseract   *ocr.Engine
	ollama      *ocr.OllamaEngine
	invertMode  ocr.InvertMode
}

func main() {
	flag.Parse()

	if *flagVerbose || *flagDebug {
		log.SetOutput(os.Stderr)
	}

	log.Println("ScreenOCR starting...")

	if err := clipboard.Init(); err != nil {
		log.Fatalf("Failed to init clipboard: %v", err)
	}

	a := &app{
		tesseract:  ocr.NewEngine(ocr.Config{Language: *flagLang, PageSegMode: *flagPSM, PreserveInterwordSpaces: true}),
		invertMode: resolveInvertMode(),
	}
	a.ollama = resolveOllama()

	handler := hk.New(hk.DefaultModifiers(), hk.DefaultKey(), a.doCapture)

	if *flagNoTray {
		runWithoutTray(handler)
	} else {
		runWithTray(handler)
	}
}

func resolveInvertMode() ocr.InvertMode {
	if *flagForceInvert {
		return ocr.InvertForce
	}
	if *flagNoInvert {
		return ocr.InvertNever
	}
	return ocr.InvertAuto
}

func resolveOllama() *ocr.OllamaEngine {
	if !*flagOllama {
		return nil
	}
	e := ocr.NewOllamaEngine(*flagOllamaModel, *flagOllamaURL)
	if e.Available() {
		log.Printf("[main] Ollama model %q is available — using as primary engine", *flagOllamaModel)
		return e
	}
	log.Println("[main] Ollama not available — using Tesseract")
	return nil
}

func (a *app) doCapture() {
	start := time.Now()
	log.Println("[main] capture workflow started")

	region, err := capture.SelectRegion()
	if err != nil {
		log.Printf("[main] selection cancelled or failed: %v", err)
		return
	}

	img, err := capture.CaptureRegion(region)
	if err != nil {
		log.Printf("[main] capture failed: %v", err)
		notify.Error("Screen capture failed")
		return
	}

	text, err := a.runOCR(img)
	if err != nil {
		log.Printf("[main] OCR failed: %v", err)
		notify.Error("OCR processing failed")
		return
	}

	if len(text) == 0 {
		notify.Info("No text detected in selection")
		return
	}

	text = a.postProcess(text)

	if err := clipboard.CopyText(text); err != nil {
		log.Printf("[main] clipboard copy failed: %v", err)
		notify.Error("Failed to copy to clipboard")
		return
	}

	elapsed := time.Since(start)
	log.Printf("[main] done in %v — %d chars extracted", elapsed, len(text))
	notify.Success(fmt.Sprintf("Copied %d chars (%dms)", len(text), elapsed.Milliseconds()))
}

func (a *app) runOCR(img image.Image) (string, error) {
	if a.ollama != nil {
		log.Println("[main] running Ollama OCR")
		text, err := a.ollama.ProcessImage(img)
		if err == nil {
			return text, nil
		}
		log.Printf("[main] Ollama failed, falling back to Tesseract: %v", err)
	}
	return a.runTesseract(img)
}

func (a *app) runTesseract(img image.Image) (string, error) {
	preprocessed := ocr.PreprocessForCode(img, a.invertMode)
	if *flagDebug {
		saveDebugImages(img, preprocessed)
	}
	return a.tesseract.ProcessImage(preprocessed)
}

func (a *app) postProcess(text string) string {
	if *flagCleanup {
		text = cleanup.CleanCodeText(text)
	}
	if *flagLineNumbers {
		text = cleanup.RemoveLineNumbers(text)
	}
	return cleanup.NormalizeIndentation(text, 4)
}

func runWithoutTray(handler *hk.Handler) {
	hk.RunOnMainThread(func() {
		ctx := context.Background()
		if err := handler.Start(ctx); err != nil {
			log.Fatalf("Failed to register hotkey: %v", err)
		}
		defer handler.Stop()
		log.Println("ScreenOCR running (Ctrl+Shift+O). Press Ctrl+C to quit.")
		select {}
	})
}

func runWithTray(handler *hk.Handler) {
	tray.Run(
		func() {
			quitCh := tray.Setup()
			ctx, cancel := context.WithCancel(context.Background())
			if err := handler.Start(ctx); err != nil {
				log.Fatalf("Failed to register hotkey: %v", err)
			}
			go func() {
				<-quitCh
				handler.Stop()
				cancel()
				os.Exit(0)
			}()
			log.Println("ScreenOCR running in system tray")
		},
		func() {
			handler.Stop()
			log.Println("ScreenOCR shutting down")
		},
	)
}

func saveDebugImages(raw image.Image, preprocessed *image.Gray) {
	dir := filepath.Join(os.TempDir(), "screenocr_debug")
	os.MkdirAll(dir, 0755)
	ts := time.Now().Format("150405")

	save := func(suffix string, img image.Image) {
		path := filepath.Join(dir, ts+suffix)
		f, err := os.Create(path)
		if err != nil {
			log.Printf("[debug] failed to create %s: %v", path, err)
			return
		}
		defer f.Close()
		png.Encode(f, img)
		log.Printf("[debug] saved %s", path)
	}

	save("_1_raw.png", raw)
	save("_2_preprocessed.png", preprocessed)
}
