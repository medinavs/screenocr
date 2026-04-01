//go:build ignore

package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"fmt"
)

func main() {
	for _, sz := range []int{32, 64} {
		img := renderIcon(sz)
		var buf bytes.Buffer
		png.Encode(&buf, img)
		name := fmt.Sprintf("internal/tray/icon%d.png", sz)
		os.WriteFile(name, buf.Bytes(), 0644)
		fmt.Printf("wrote %s (%d bytes)\n", name, buf.Len())
	}
}

func renderIcon(sz int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, sz, sz))
	bg := color.NRGBA{R: 26, G: 32, B: 44, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	applyRoundedMask(img, sz, int(math.Round(float64(sz)*0.18)))
	accent := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	pad := max1(int(math.Round(float64(sz)*0.18)), 2)
	thick := max1(int(math.Round(float64(sz)*0.09)), 1)
	arm := max1(int(math.Round(float64(sz)*0.22)), 2)
	drawBrackets(img, pad, pad, sz-pad, sz-pad, arm, thick, accent)
	cx, cy := sz/2, sz/2
	dot := max1(int(math.Round(float64(sz)*0.07)), 1)
	fillRect(img, cx-dot, cy-dot, cx+dot, cy+dot, color.NRGBA{R: 80, G: 200, B: 255, A: 220})
	return img
}

func drawBrackets(img *image.NRGBA, x1, y1, x2, y2, arm, thick int, c color.NRGBA) {
	fillRect(img, x1, y1, x1+arm, y1+thick, c)
	fillRect(img, x1, y1, x1+thick, y1+arm, c)
	fillRect(img, x2-arm, y1, x2, y1+thick, c)
	fillRect(img, x2-thick, y1, x2, y1+arm, c)
	fillRect(img, x1, y2-thick, x1+arm, y2, c)
	fillRect(img, x1, y2-arm, x1+thick, y2, c)
	fillRect(img, x2-arm, y2-thick, x2, y2, c)
	fillRect(img, x2-thick, y2-arm, x2, y2, c)
}

func fillRect(img *image.NRGBA, x1, y1, x2, y2 int, c color.NRGBA) {
	b := img.Bounds()
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			if x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func applyRoundedMask(img *image.NRGBA, sz, r int) {
	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			if !insideRoundedRect(x, y, sz, r) {
				img.SetNRGBA(x, y, color.NRGBA{})
			}
		}
	}
}

func insideRoundedRect(x, y, sz, r int) bool {
	cx1, cy1 := r, r
	cx2, cy2 := sz-r-1, sz-r-1
	if x >= cx1 && x <= cx2 && y >= 0 && y < sz {
		return true
	}
	if y >= cy1 && y <= cy2 && x >= 0 && x < sz {
		return true
	}
	for _, c := range [][2]int{{cx1, cy1}, {cx2, cy1}, {cx1, cy2}, {cx2, cy2}} {
		dx, dy := float64(x-c[0]), float64(y-c[1])
		if math.Sqrt(dx*dx+dy*dy) <= float64(r) {
			return true
		}
	}
	return false
}

func max1(a, b int) int {
	if a > b {
		return a
	}
	return b
}
