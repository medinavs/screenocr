package ocr

import (
	"image"
	"image/color"
	"log"
	"math"
)

// InvertMode controls background inversion behaviour.
type InvertMode int

const (
	InvertAuto  InvertMode = iota // detect automatically (default)
	InvertForce                   // always invert
	InvertNever                   // never invert
)

// PreprocessForCode applies the full preprocessing pipeline.
// invertMode lets callers override the auto-detection (useful for VDI / dark themes).
func PreprocessForCode(img image.Image, invertMode InvertMode) *image.Gray {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	log.Printf("[ocr/preprocess] input: %dx%d", w, h)

	gray := toGrayscale(img)

	// Invert step — must happen before binarization
	switch invertMode {
	case InvertForce:
		log.Println("[ocr/preprocess] force-invert enabled")
		gray = invertImage(gray)
	case InvertNever:
		log.Println("[ocr/preprocess] invert disabled")
	default:
		gray = InvertIfDarkBackground(gray)
	}

	// Scale — Tesseract needs ~32px tall text
	switch {
	case h < 400:
		gray = scaleUp(gray, 4)
	case h < 800:
		gray = scaleUp(gray, 3)
	default:
		gray = scaleUp(gray, 2)
	}
	log.Printf("[ocr/preprocess] after scale: %dx%d", gray.Bounds().Dx(), gray.Bounds().Dy())

	// Denoise BEFORE sharpening — removes RDP/VDI compression block artifacts
	// that would otherwise be amplified by the sharpen kernel.
	gray = boxBlur(gray, 1)

	// Sharpen to restore edge crispness after blur
	gray = sharpen(gray)

	gray = stretchContrast(gray)
	gray = otsuThreshold(gray)

	log.Printf("[ocr/preprocess] output: %dx%d", gray.Bounds().Dx(), gray.Bounds().Dy())
	return gray
}

func toGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			gray.SetGray(x, y, color.Gray{Y: uint8(lum / 256)})
		}
	}
	return gray
}

// InvertIfDarkBackground detects a dark background and inverts so text is dark-on-light.
func InvertIfDarkBackground(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	pixels := bounds.Dx() * bounds.Dy()

	// Count dark vs light pixels (more reliable than mean on VDI compressed images)
	var dark int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.GrayAt(x, y).Y < 128 {
				dark++
			}
		}
	}
	ratio := float64(dark) / float64(pixels)

	if ratio > 0.55 {
		log.Printf("[ocr/preprocess] dark background (dark ratio=%.2f), inverting", ratio)
		return invertImage(img)
	}
	return img
}

func invertImage(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	result := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			result.SetGray(x, y, color.Gray{Y: 255 - img.GrayAt(x, y).Y})
		}
	}
	return result
}

func scaleUp(img *image.Gray, factor int) *image.Gray {
	bounds := img.Bounds()
	newW := bounds.Dx() * factor
	newH := bounds.Dy() * factor
	scaled := image.NewGray(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			scaled.SetGray(x, y, img.GrayAt(bounds.Min.X+x/factor, bounds.Min.Y+y/factor))
		}
	}
	return scaled
}

// boxBlur smooths the image with a (2r+1)×(2r+1) box kernel.
// Radius 1 → 3×3 kernel, which removes RDP block artifacts without blurring text.
func boxBlur(img *image.Gray, radius int) *image.Gray {
	bounds := img.Bounds()
	result := image.NewGray(bounds)
	size := (2*radius + 1) * (2*radius + 1)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var sum int
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					nx := x + dx
					ny := y + dy
					if nx < bounds.Min.X {
						nx = bounds.Min.X
					} else if nx >= bounds.Max.X {
						nx = bounds.Max.X - 1
					}
					if ny < bounds.Min.Y {
						ny = bounds.Min.Y
					} else if ny >= bounds.Max.Y {
						ny = bounds.Max.Y - 1
					}
					sum += int(img.GrayAt(nx, ny).Y)
				}
			}
			result.SetGray(x, y, color.Gray{Y: uint8(sum / size)})
		}
	}
	return result
}

// sharpen applies a Laplacian sharpen kernel to make character edges crisper.
func sharpen(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	result := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if x == bounds.Min.X || x == bounds.Max.X-1 ||
				y == bounds.Min.Y || y == bounds.Max.Y-1 {
				result.SetGray(x, y, img.GrayAt(x, y))
				continue
			}
			val := 5*int(img.GrayAt(x, y).Y) -
				int(img.GrayAt(x-1, y).Y) -
				int(img.GrayAt(x+1, y).Y) -
				int(img.GrayAt(x, y-1).Y) -
				int(img.GrayAt(x, y+1).Y)
			if val < 0 {
				val = 0
			} else if val > 255 {
				val = 255
			}
			result.SetGray(x, y, color.Gray{Y: uint8(val)})
		}
	}
	return result
}

func stretchContrast(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	var minVal uint8 = 255
	var maxVal uint8 = 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			v := img.GrayAt(x, y).Y
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal == minVal {
		return img
	}
	result := image.NewGray(bounds)
	spread := float64(maxVal - minVal)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			v := img.GrayAt(x, y).Y
			result.SetGray(x, y, color.Gray{Y: uint8(math.Round(float64(v-minVal) / spread * 255))})
		}
	}
	return result
}

func otsuThreshold(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	total := bounds.Dx() * bounds.Dy()

	var hist [256]int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			hist[img.GrayAt(x, y).Y]++
		}
	}

	var sumTotal float64
	for i := 0; i < 256; i++ {
		sumTotal += float64(i) * float64(hist[i])
	}

	var sumBg float64
	var wBg int
	var maxVar float64
	var threshold uint8

	for t := 0; t < 256; t++ {
		wBg += hist[t]
		if wBg == 0 {
			continue
		}
		wFg := total - wBg
		if wFg == 0 {
			break
		}
		sumBg += float64(t) * float64(hist[t])
		mBg := sumBg / float64(wBg)
		mFg := (sumTotal - sumBg) / float64(wFg)
		v := float64(wBg) * float64(wFg) * (mBg - mFg) * (mBg - mFg)
		if v > maxVar {
			maxVar = v
			threshold = uint8(t)
		}
	}

	log.Printf("[ocr/preprocess] Otsu threshold: %d", threshold)
	result := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.GrayAt(x, y).Y > threshold {
				result.SetGray(x, y, color.Gray{Y: 255})
			} else {
				result.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}
	return result
}
