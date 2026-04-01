//go:build windows

package capture

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procSetCursor           = user32.NewProc("SetCursor")
	procSetCapture          = user32.NewProc("SetCapture")
	procReleaseCapture      = user32.NewProc("ReleaseCapture")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procUpdateLayeredWindow = user32.NewProc("UpdateLayeredWindow")
	procUnregisterClassW    = user32.NewProc("UnregisterClassW")
	procSetProcessDPIAware  = user32.NewProc("SetProcessDPIAware")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCxVirtualScreen = 78
	smCyVirtualScreen = 79
	vkEscape          = 0x1B

	wsExLayered    = 0x00080000
	wsExTopmost    = 0x00000008
	wsExToolwindow = 0x00000080
	wsPopup        = 0x80000000

	ulwAlpha = 0x00000002
	swShow   = 5
	idcCross = 32515

	wmDestroy     = 0x0002
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmMouseMove   = 0x0200
	wmKeyDown     = 0x0100
	wmRButtonDown = 0x0204

	overlayAlpha = 160 // darkness of the surrounding dim (0-255)
	borderWidth  = 2   // selection border thickness in pixels

	// BGRA premultiplied pixels
	darkPixel   = uint32(overlayAlpha) << 24                                           // black, semi-transparent
	borderPixel = uint32(0xFF) | uint32(0xFF)<<8 | uint32(0xFF)<<16 | uint32(0xFF)<<24 // white, opaque
)

type point struct{ X, Y int32 }
type ptSize struct{ CX, CY int32 }

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	_      [4]byte
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type msgStruct struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type overlayState struct {
	startPt   point
	endPt     point
	dragging  bool
	cancelled bool
	originX   int32
	originY   int32
	screenW   int32
	screenH   int32
}

var (
	ov              overlayState
	wndProcCallback uintptr
)

func init() {
	wndProcCallback = syscall.NewCallback(overlayWndProc)
	procSetProcessDPIAware.Call()
}

func loword(l uintptr) int32 { return int32(int16(l & 0xFFFF)) }
func hiword(l uintptr) int32 { return int32(int16((l >> 16) & 0xFFFF)) }

func clamp32(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func overlayWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmLButtonDown:
		ov.startPt = point{loword(lParam), hiword(lParam)}
		ov.endPt = ov.startPt
		ov.dragging = true
		procSetCapture.Call(hwnd)
		paintOverlay(hwnd)
		return 0

	case wmMouseMove:
		if ov.dragging {
			ov.endPt = point{loword(lParam), hiword(lParam)}
			paintOverlay(hwnd)
		}
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(idcCross))
		procSetCursor.Call(cursor)
		return 0

	case wmLButtonUp:
		if ov.dragging {
			ov.endPt = point{loword(lParam), hiword(lParam)}
			ov.dragging = false
			procReleaseCapture.Call()
			procPostQuitMessage.Call(0)
		}
		return 0

	case wmKeyDown:
		if wParam == vkEscape {
			ov.cancelled = true
			procReleaseCapture.Call()
			procPostQuitMessage.Call(0)
		}
		return 0

	case wmRButtonDown:
		ov.cancelled = true
		procReleaseCapture.Call()
		procPostQuitMessage.Call(0)
		return 0

	case wmDestroy:
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

// paintOverlay redraws the layered window using per-pixel alpha:
//   - dark semi-transparent pixels outside the selection
//   - transparent (alpha=0) pixels inside  → screen shows through
//   - white opaque pixels for the border
func paintOverlay(hwnd uintptr) {
	w, h := ov.screenW, ov.screenH

	screenDC, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screenDC)

	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	defer procDeleteDC.Call(memDC)

	bi := bitmapInfo{
		Header: bitmapInfoHeader{
			Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			Width:    w,
			Height:   -h, // top-down raster
			Planes:   1,
			BitCount: 32,
		},
	}

	var pixPtr uintptr
	hbm, _, _ := procCreateDIBSection.Call(
		screenDC,
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
		uintptr(unsafe.Pointer(&pixPtr)),
		0, 0,
	)
	if hbm == 0 {
		return
	}
	defer procDeleteObject.Call(hbm)

	oldBm, _, _ := procSelectObject.Call(memDC, hbm)
	defer procSelectObject.Call(memDC, oldBm)

	nPixels := int(w) * int(h)
	pixels := unsafe.Slice((*uint32)(unsafe.Pointer(pixPtr)), nPixels)

	// overlay
	for i := range pixels {
		pixels[i] = darkPixel
	}

	if ov.dragging {
		x1 := clamp32(ov.startPt.X, 0, w)
		y1 := clamp32(ov.startPt.Y, 0, h)
		x2 := clamp32(ov.endPt.X, 0, w)
		y2 := clamp32(ov.endPt.Y, 0, h)
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		if y1 > y2 {
			y1, y2 = y2, y1
		}

		if x2-x1 >= 2 && y2-y1 >= 2 {
			for y := y1 + borderWidth; y < y2-borderWidth; y++ {
				row := pixels[int(y)*int(w):]
				for x := x1 + borderWidth; x < x2-borderWidth; x++ {
					row[x] = 0
				}
			}
			// top
			for y := y1; y < y1+borderWidth && y < y2; y++ {
				row := pixels[int(y)*int(w):]
				for x := x1; x < x2; x++ {
					row[x] = borderPixel
				}
			}
			// bottom
			for y := y2 - borderWidth; y < y2 && y >= y1+borderWidth; y++ {
				row := pixels[int(y)*int(w):]
				for x := x1; x < x2; x++ {
					row[x] = borderPixel
				}
			}
			// left
			for y := y1 + borderWidth; y < y2-borderWidth; y++ {
				row := pixels[int(y)*int(w):]
				for x := x1; x < x1+borderWidth; x++ {
					row[x] = borderPixel
				}
			}
			// right
			for y := y1 + borderWidth; y < y2-borderWidth; y++ {
				row := pixels[int(y)*int(w):]
				for x := x2 - borderWidth; x < x2; x++ {
					row[x] = borderPixel
				}
			}
		}
	}

	// BLENDFUNCTION: BlendOp=AC_SRC_OVER(0), BlendFlags=0, ConstAlpha=255, AlphaFormat=AC_SRC_ALPHA(1)
	var blend [4]byte = [4]byte{0, 0, 255, 1}
	ptDst := point{ov.originX, ov.originY}
	sz := ptSize{w, h}
	ptSrc := point{0, 0}

	procUpdateLayeredWindow.Call(
		hwnd,
		screenDC,
		uintptr(unsafe.Pointer(&ptDst)),
		uintptr(unsafe.Pointer(&sz)),
		memDC,
		uintptr(unsafe.Pointer(&ptSrc)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ulwAlpha,
	)
}

var overlayClassName, _ = syscall.UTF16PtrFromString("ScreenOCROverlay")

func selectRegion() (Region, error) {
	log.Println("[capture] starting overlay region selection")

	vx, _, _ := procGetSystemMetrics.Call(uintptr(smXVirtualScreen))
	vy, _, _ := procGetSystemMetrics.Call(uintptr(smYVirtualScreen))
	vw, _, _ := procGetSystemMetrics.Call(uintptr(smCxVirtualScreen))
	vh, _, _ := procGetSystemMetrics.Call(uintptr(smCyVirtualScreen))

	ov = overlayState{
		originX: int32(int(vx)),
		originY: int32(int(vy)),
		screenW: int32(vw),
		screenH: int32(vh),
	}

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(idcCross))

	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   wndProcCallback,
		Instance:  hInstance,
		Cursor:    cursor,
		ClassName: overlayClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWindowExW.Call(
		wsExLayered|wsExTopmost|wsExToolwindow,
		uintptr(unsafe.Pointer(overlayClassName)),
		0,
		wsPopup,
		uintptr(uint32(ov.originX)), uintptr(uint32(ov.originY)),
		uintptr(vw), uintptr(vh),
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return Region{}, fmt.Errorf("failed to create overlay window")
	}

	paintOverlay(hwnd)
	procShowWindow.Call(hwnd, swShow)
	procSetForegroundWindow.Call(hwnd)

	var m msgStruct
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if ret == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	procDestroyWindow.Call(hwnd)
	procUnregisterClassW.Call(uintptr(unsafe.Pointer(overlayClassName)), hInstance)

	if ov.cancelled {
		return Region{}, fmt.Errorf("selection cancelled")
	}

	x1 := ov.startPt.X + ov.originX
	y1 := ov.startPt.Y + ov.originY
	x2 := ov.endPt.X + ov.originX
	y2 := ov.endPt.Y + ov.originY

	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}

	w, h := int(x2-x1), int(y2-y1)
	if w < 5 || h < 5 {
		return Region{}, fmt.Errorf("selected region too small: %dx%d", w, h)
	}

	log.Printf("[capture] region selected: (%d,%d) %dx%d", x1, y1, w, h)
	return Region{X: int(x1), Y: int(y1), W: w, H: h}, nil
}
