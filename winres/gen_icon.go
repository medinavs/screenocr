//go:build ignore

// Run with: go run winres/gen_icon.go
// Generates winres/icon.ico used by go-winres to embed into the .exe

package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

func main() {
	sizes := []int{256, 64, 48, 32, 16}
	var pngs [][]byte
	for _, sz := range sizes {
		pngs = append(pngs, renderIcon(sz))
	}
	writeICO("winres/icon.ico", pngs, sizes)
}

// renderIcon draws the ScreenOCR icon at the given size:
// Dark rounded square + white selection bracket corners + cursor dot
func renderIcon(sz int) []byte {
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

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func drawBrackets(img *image.NRGBA, x1, y1, x2, y2, arm, thick int, c color.NRGBA) {
	// Top-left
	fillRect(img, x1, y1, x1+arm, y1+thick, c)
	fillRect(img, x1, y1, x1+thick, y1+arm, c)
	// Top-right
	fillRect(img, x2-arm, y1, x2, y1+thick, c)
	fillRect(img, x2-thick, y1, x2, y1+arm, c)
	// Bottom-left
	fillRect(img, x1, y2-thick, x1+arm, y2, c)
	fillRect(img, x1, y2-arm, x1+thick, y2, c)
	// Bottom-right
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
	corners := [][2]int{{cx1, cy1}, {cx2, cy1}, {cx1, cy2}, {cx2, cy2}}
	for _, c := range corners {
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

// writeICO writes a simple ICO file from a slice of PNG-encoded images.
func writeICO(path string, pngs [][]byte, sizes []int) {
	f, _ := os.Create(path)
	defer f.Close()

	n := len(pngs)
	// Header: reserved(2) + type(2) + count(2)
	binary.Write(f, binary.LittleEndian, uint16(0))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(n))

	// Directory entries (16 bytes each)
	offset := uint32(6 + 16*n)
	for i, p := range pngs {
		sz := sizes[i]
		w, h := uint8(sz), uint8(sz)
		if sz >= 256 {
			w, h = 0, 0
		}
		f.Write([]byte{w, h, 0, 0})
		binary.Write(f, binary.LittleEndian, uint16(1))
		binary.Write(f, binary.LittleEndian, uint16(32))
		binary.Write(f, binary.LittleEndian, uint32(len(p)))
		binary.Write(f, binary.LittleEndian, offset)
		offset += uint32(len(p))
	}

	for _, p := range pngs {
		f.Write(p)
	}
}
