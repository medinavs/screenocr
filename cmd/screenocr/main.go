package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
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
)

func main() {
	flag.Parse()

	if !*flagVerbose {
		log.SetOutput(os.Stderr)
	}

	log.Println("ScreenOCR starting...")

	if err := clipboard.Init(); err != nil {
		log.Fatalf("Failed to init clipboard: %v", err)
	}

	tesseract := ocr.NewEngine(ocr.Config{
		Language:                *flagLang,
		PageSegMode:             *flagPSM,
		PreserveInterwordSpaces: true,
	})

	var ollamaEngine *ocr.OllamaEngine
	if *flagOllama {
		ollamaEngine = ocr.NewOllamaEngine(*flagOllamaModel, *flagOllamaURL)
		if ollamaEngine.Available() {
			log.Printf("[main] Ollama model %q is available — using as primary engine", *flagOllamaModel)
		} else {
			log.Printf("[main] Ollama not available — using Tesseract")
			ollamaEngine = nil
		}
	}

	doCapture := func() {
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

		var text string

		if ollamaEngine != nil {
			// Ollama receives the raw screenshot — no preprocessing needed
			log.Println("[main] running Ollama OCR")
			text, err = ollamaEngine.ProcessImage(img)
			if err != nil {
				log.Printf("[main] Ollama failed, falling back to Tesseract: %v", err)
				preprocessed := ocr.PreprocessForCode(img)
				text, err = tesseract.ProcessImage(preprocessed)
			}
		} else {
			preprocessed := ocr.PreprocessForCode(img)
			text, err = tesseract.ProcessImage(preprocessed)
		}

		if err != nil {
			log.Printf("[main] OCR failed: %v", err)
			notify.Error("OCR processing failed")
			return
		}

		if len(text) == 0 {
			notify.Info("No text detected in selection")
			return
		}

		if *flagCleanup {
			text = cleanup.CleanCodeText(text)
		}
		if *flagLineNumbers {
			text = cleanup.RemoveLineNumbers(text)
		}
		text = cleanup.NormalizeIndentation(text, 4)

		if err := clipboard.CopyText(text); err != nil {
			log.Printf("[main] clipboard copy failed: %v", err)
			notify.Error("Failed to copy to clipboard")
			return
		}

		elapsed := time.Since(start)
		log.Printf("[main] done in %v — %d chars extracted", elapsed, len(text))
		notify.Success(fmt.Sprintf("Copied %d chars (%dms)", len(text), elapsed.Milliseconds()))
	}

	handler := hk.New(hk.DefaultModifiers(), hk.DefaultKey(), doCapture)

	if *flagNoTray {
		hk.RunOnMainThread(func() {
			ctx := context.Background()
			if err := handler.Start(ctx); err != nil {
				log.Fatalf("Failed to register hotkey: %v", err)
			}
			defer handler.Stop()
			log.Println("ScreenOCR running (Ctrl+Shift+O). Press Ctrl+C to quit.")
			select {}
		})
	} else {
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
}
