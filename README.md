# ScreenOCR

Press a hotkey, drag to select a screen region, and the extracted text is instantly in your clipboard. Built for code screenshots — runs entirely offline.

---

## Quick Start (Windows)

**1. Install Tesseract**
Download from https://github.com/UB-Mannheim/tesseract/wiki and add to PATH:
```powershell
$env:PATH += ";C:\Program Files\Tesseract-OCR"
```

**2. Build and run**
```powershell
go build -o screenocr.exe ./cmd/screenocr
./screenocr.exe
```

**3. Use it**
- Press **Ctrl+Shift+O**
- Drag to select the area you want to OCR
- Text is copied to clipboard automatically

---

## How It Works

```
Ctrl+Shift+O
    │
    ▼
Overlay appears — drag to select region
    │
    ▼
Screenshot of selected pixels captured
    │
    ▼
Preprocessed: invert dark bg → scale up → sharpen → binarize
    │
    ▼  (if Ollama available, skips preprocessing)
Tesseract CLI  ──or──  Ollama vision model
    │
    ▼
Cleanup: fix OCR errors, strip line numbers, normalize indentation
    │
    ▼
Copied to clipboard + system notification
```

---

## Ollama (Better Accuracy)

If [Ollama](https://ollama.com) is running locally, ScreenOCR uses it automatically as the primary OCR engine — significantly better accuracy than Tesseract, especially for code.

```powershell
# install a vision model (pick one)
ollama pull moondream    # 1.8 GB — fast
ollama pull minicpm-v    # 5 GB  — best accuracy
ollama pull llava        # 4 GB  — good balance

# ScreenOCR auto-detects; no extra config needed
./screenocr.exe

# force Tesseract only
./screenocr.exe -ollama=false

# use a specific model
./screenocr.exe -ollama-model=minicpm-v
```

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-cleanup` | `true` | Fix common OCR errors in code |
| `-strip-linenum` | `true` | Remove editor line numbers |
| `-psm` | `6` | Tesseract page segmentation mode |
| `-lang` | `eng` | Tesseract language |
| `-no-tray` | `false` | Run without system tray icon |
| `-verbose` | `false` | Enable verbose logging |
| `-ollama` | `true` | Use Ollama when available |
| `-ollama-model` | `moondream` | Ollama model name |
| `-ollama-url` | `http://localhost:11434` | Ollama API URL |

---

## Building

```powershell
# dev
go run ./cmd/screenocr -verbose -no-tray

# prod build (with icon + manifest embedded)
make build-windows        # → build/windows/screenocr.exe

# other platforms
make build-linux          # → build/linux/screenocr
make build-darwin         # → build/darwin/screenocr
```

### Regenerating the Icon

```powershell
go run winres/gen_icon.go   # regenerates winres/icon.ico
make winres                 # rebuilds the .syso resource files
```

---

## Project Structure

```
cmd/screenocr/
  main.go                    ← entry point
  rsrc_windows_amd64.syso    ← embedded icon + manifest (auto-generated)

internal/
  capture/
    capture.go               ← screenshot capture (kbinani/screenshot)
    select_windows.go        ← Win32 overlay with per-pixel alpha
    select_darwin.go         ← macOS screencapture
    select_linux.go          ← slop / xdotool

  ocr/
    ocr.go                   ← Tesseract CLI engine
    ollama.go                ← Ollama vision model engine
    preprocess.go            ← grayscale → scale → sharpen → Otsu binarize

  hotkey/
    hotkey.go                ← global hotkey (golang.design/x/hotkey)
    modifiers_*.go           ← per-platform Ctrl/Cmd+Shift constants

  cleanup/
    cleanup.go               ← keyword fixes, line number stripping, indentation
    cleanup_test.go

  clipboard/
    clipboard.go             ← golang.design/x/clipboard

  notify/
    notify.go                ← system notifications (gen2brain/beeep)

  tray/
    tray.go                  ← system tray (getlantern/systray, Windows)
    tray_other.go            ← no-op stub for non-Windows

winres/
  gen_icon.go                ← icon generator (go run winres/gen_icon.go)
  icon.ico                   ← generated icon (256/64/48/32/16px)
  winres.json                ← go-winres manifest
```

---

## Tesseract PSM Modes

| PSM | Best For |
|-----|----------|
| `3` | General / mixed content |
| `4` | Single column, variable font sizes |
| **`6`** | **Code blocks (default)** |
| `7` | Single line |
| `11` | Sparse text / UI labels |

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `tesseract` not found | Add `C:\Program Files\Tesseract-OCR` to PATH, open a new terminal |
| Wrong region captured | Check display scaling — app calls `SetProcessDPIAware` automatically |
| Missing text in output | Try `-psm 3` or `-psm 4`; ensure selection is tight around text |
| Dark theme OCR fails | Preprocessor inverts dark backgrounds automatically; try `-cleanup=false` to debug |
| Overlay crashes on edges | Update to latest build — fixed via `clamp32` in pixel buffer writes |

---

## Prerequisites

### Windows
- Go 1.21+
- [Tesseract OCR](https://github.com/UB-Mannheim/tesseract/wiki) in PATH
- No GCC / CGO required

### Linux
```bash
sudo apt-get install -y tesseract-ocr tesseract-ocr-eng \
  libx11-dev libxrandr-dev slop gcc
```

### macOS
```bash
brew install tesseract pkg-config
```

---

## License

MIT
