//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")
	kern32 = syscall.NewLazyDLL("kernel32.dll")
	winmm  = syscall.NewLazyDLL("winmm.dll")

	pRegisterClassEx  = user32.NewProc("RegisterClassExW")
	pCreateWindowEx   = user32.NewProc("CreateWindowExW")
	pDefWindowProc    = user32.NewProc("DefWindowProcW")
	pDestroyWindow    = user32.NewProc("DestroyWindow")
	pPeekMessage      = user32.NewProc("PeekMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pDispatchMessage  = user32.NewProc("DispatchMessageW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pGetDC            = user32.NewProc("GetDC")
	pGetClientRect    = user32.NewProc("GetClientRect")
	pAdjustWindowRect = user32.NewProc("AdjustWindowRectEx")
	pLoadCursor       = user32.NewProc("LoadCursorW")
	pGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	pSetDPIAware      = user32.NewProc("SetProcessDPIAware")
	pMessageBox       = user32.NewProc("MessageBoxW")
	pGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	pGetForeground    = user32.NewProc("GetForegroundWindow")
	pEnumMonitors     = user32.NewProc("EnumDisplayMonitors")
	pGetWindowRect    = user32.NewProc("GetWindowRect")
	pReleaseDC        = user32.NewProc("ReleaseDC")
	pStretchDIBits    = gdi32.NewProc("StretchDIBits")
	pSetStretchMode   = gdi32.NewProc("SetStretchBltMode")
	pPatBlt           = gdi32.NewProc("PatBlt")
	pGetModuleHandle  = kern32.NewProc("GetModuleHandleW")
	pTimeBeginPeriod  = winmm.NewProc("timeBeginPeriod")
	pWaveOutOpen      = winmm.NewProc("waveOutOpen")
	pWaveOutPrepare   = winmm.NewProc("waveOutPrepareHeader")
	pWaveOutWrite     = winmm.NewProc("waveOutWrite")
)

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

type msgT struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	X, Y    int32
	Private uint32
}

type rectT struct{ Left, Top, Right, Bottom int32 }

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

type waveFormatEx struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	Size           uint16
}

type waveHdr struct {
	Data          uintptr
	BufferLength  uint32
	BytesRecorded uint32
	User          uintptr
	Flags         uint32
	Loops         uint32
	Next          uintptr
	Reserved      uintptr
}

const audioBufs, audioLen = 3, 1024

var (
	waveHdrs [audioBufs]waveHdr
	waveData [audioBufs][audioLen]int16
	waveFmt  = waveFormatEx{1, 1, SampleRate, SampleRate * 2, 2, 16, 0}
)

// startAudio streams the synth through winmm. Any failure just means silence.
func startAudio(a *Audio) {
	var h uintptr
	r, _, _ := pWaveOutOpen.Call(uintptr(unsafe.Pointer(&h)), 0xFFFFFFFF,
		uintptr(unsafe.Pointer(&waveFmt)), 0, 0, 0)
	if r != 0 || h == 0 {
		return
	}
	for i := range waveHdrs {
		waveHdrs[i].Data = uintptr(unsafe.Pointer(&waveData[i][0]))
		waveHdrs[i].BufferLength = audioLen * 2
		r, _, _ = pWaveOutPrepare.Call(h, uintptr(unsafe.Pointer(&waveHdrs[i])), unsafe.Sizeof(waveHdrs[i]))
		if r != 0 {
			return
		}
	}
	go func() {
		first := true
		for {
			for i := range waveHdrs {
				if first || atomic.LoadUint32(&waveHdrs[i].Flags)&1 != 0 { // WHDR_DONE
					a.Fill(waveData[i][:])
					pWaveOutWrite.Call(h, uintptr(unsafe.Pointer(&waveHdrs[i])), unsafe.Sizeof(waveHdrs[i]))
				}
			}
			first = false
			time.Sleep(4 * time.Millisecond)
		}
	}()
}

const (
	wmDestroy    = 0x0002
	wmKillFocus  = 0x0008
	wmClose      = 0x0010
	wmQuit       = 0x0012
	wmEraseBkgnd = 0x0014
	wmSysKeyDown = 0x0104
)

// Keys are read with GetAsyncKeyState every frame rather than from window
// messages, so that with one window per player every player can drive no
// matter which of the game's windows has the focus.
var (
	keys       [256]bool
	mainWnd    uintptr
	secondWnd  uintptr
	secondShut bool // the player closed the second window
)

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmClose:
		if hwnd == secondWnd { // closing the extra window goes back to one
			secondShut = true
			return 0
		}
	case wmDestroy:
		if hwnd == mainWnd {
			pPostQuitMessage.Call(0)
		}
		return 0
	case wmSysKeyDown:
		if wp == 0x79 || wp == 0x12 { // F10 and Alt alone would open the window menu and freeze input
			return 0
		}
	case wmEraseBkgnd:
		return 1
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
	return r
}

func pollKeys() {
	fg, _, _ := pGetForeground.Call()
	if fg == 0 || fg != mainWnd && fg != secondWnd {
		keys = [256]bool{}
		return
	}
	for k := 7; k < 256; k++ { // skip the mouse buttons
		r, _, _ := pGetAsyncKeyState.Call(uintptr(k))
		keys[k] = r&0x8000 != 0
	}
}

func init() { runtime.LockOSThread() }

func fatal(s string) {
	os.WriteFile(fileDir+"combine_baler_crash.txt", []byte(s), 0644)
	t, _ := syscall.UTF16PtrFromString("Combine-Baler")
	m, _ := syscall.UTF16PtrFromString(s)
	pMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x10)
	os.Exit(1)
}

const wsOverlapped = 0x00CF0000 // WS_OVERLAPPEDWINDOW

var (
	hInst     uintptr
	className *uint16
	scale     = 1
)

// monitors lists the attached screens. The callback is made once: Windows
// callbacks are never freed.
var (
	monList []rectT
	monCB   = syscall.NewCallback(func(hmon, hdc uintptr, rc *rectT, data uintptr) uintptr {
		monList = append(monList, *rc)
		return 1
	})
)

func monitors() []rectT {
	monList = monList[:0]
	pEnumMonitors.Call(0, 0, monCB, 0)
	return monList
}

// openWindow makes a resizable window centred on the given screen area.
func openWindow(title string, area rectT) uintptr {
	t, _ := syscall.UTF16PtrFromString(title)
	rc := rectT{0, 0, int32(SW * scale), int32(SH * scale)}
	pAdjustWindowRect.Call(uintptr(unsafe.Pointer(&rc)), wsOverlapped, 0, 0)
	ww, wh := rc.Right-rc.Left, rc.Bottom-rc.Top
	wx := area.Left + max(0, (area.Right-area.Left-ww)/2)
	wy := area.Top + max(0, (area.Bottom-area.Top-wh)/2)
	hwnd, _, err := pCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(t)),
		wsOverlapped|0x10000000, // | WS_VISIBLE
		uintptr(wx), uintptr(wy), uintptr(ww), uintptr(wh), 0, 0, hInst, 0)
	if hwnd == 0 {
		fatal(fmt.Sprintf("CreateWindowEx failed: %v", err))
	}
	hdc, _, _ := pGetDC.Call(hwnd)
	pSetStretchMode.Call(hdc, 3) // COLORONCOLOR: crisp pixels
	pReleaseDC.Call(hwnd, hdc)
	return hwnd
}

// secondArea is where the second window opens: the middle of another
// monitor if there is one, else beside the first window.
func secondArea() rectT {
	var r1 rectT
	pGetWindowRect.Call(mainWnd, uintptr(unsafe.Pointer(&r1)))
	cx, cy := (r1.Left+r1.Right)/2, (r1.Top+r1.Bottom)/2
	for _, m := range monitors() {
		if cx < m.Left || cx >= m.Right || cy < m.Top || cy >= m.Bottom {
			return m
		}
	}
	w := r1.Right - r1.Left
	return rectT{r1.Left + w/3, r1.Top + 40, r1.Right + w/3, r1.Bottom + 40}
}

var bmi = bitmapInfoHeader{Width: SW, Height: -SH, Planes: 1, BitCount: 32}

// blit scales a framebuffer into a window, letterboxed, whole pixels when
// it fits.
func blit(hwnd uintptr, fb []uint32) {
	hdc, _, _ := pGetDC.Call(hwnd)
	defer pReleaseDC.Call(hwnd, hdc)
	pSetStretchMode.Call(hdc, 3)
	var cr rectT
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
	cw, ch := int(cr.Right), int(cr.Bottom)
	if cw <= 0 || ch <= 0 {
		return
	}
	dw, dh := cw, cw*SH/SW
	if dh > ch {
		dw, dh = ch*SW/SH, ch
	}
	if dw >= SW && dh >= SH { // snap to whole pixels when possible
		s := min(dw/SW, dh/SH)
		dw, dh = SW*s, SH*s
	}
	dx, dy := (cw-dw)/2, (ch-dh)/2
	const blackness = 0x42
	if dx > 0 {
		pPatBlt.Call(hdc, 0, 0, uintptr(dx), uintptr(ch), blackness)
		pPatBlt.Call(hdc, uintptr(dx+dw), 0, uintptr(cw-dx-dw), uintptr(ch), blackness)
	}
	if dy > 0 {
		pPatBlt.Call(hdc, 0, 0, uintptr(cw), uintptr(dy), blackness)
		pPatBlt.Call(hdc, 0, uintptr(dy+dh), uintptr(cw), uintptr(ch-dy-dh), blackness)
	}
	pStretchDIBits.Call(hdc,
		uintptr(dx), uintptr(dy), uintptr(dw), uintptr(dh),
		0, 0, SW, SH,
		uintptr(unsafe.Pointer(&fb[0])), uintptr(unsafe.Pointer(&bmi)),
		0, 0x00CC0020) // DIB_RGB_COLORS, SRCCOPY
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fatal(fmt.Sprintf("Combine-Baler crashed: %v\n\n%s", r, debug.Stack()))
		}
	}()
	pSetDPIAware.Call()
	pTimeBeginPeriod.Call(1)

	hInst, _, _ = pGetModuleHandle.Call(0)
	className, _ = syscall.UTF16PtrFromString("CombineBalerWindow")
	cursor, _, _ := pLoadCursor.Call(0, 32512)

	wc := wndClassEx{
		Style:     0x0003, // CS_HREDRAW | CS_VREDRAW
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  hInst,
		Cursor:    cursor,
		ClassName: className,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if r, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		fatal(fmt.Sprintf("RegisterClassEx failed: %v", err))
	}

	// biggest whole-number scale that fits comfortably on the screen
	scrW, _, _ := pGetSystemMetrics.Call(0)
	scrH, _, _ := pGetSystemMetrics.Call(1)
	for (scale+1)*SW <= int(scrW)*9/10 && (scale+1)*SH <= int(scrH)*85/100 {
		scale++
	}
	mainWnd = openWindow("Combine-Baler", rectT{0, 0, int32(scrW), int32(scrH)})
	bmi.Size = uint32(unsafe.Sizeof(bmi))

	g := NewGame()
	startAudio(g.Snd)
	const step = 1.0 / 60
	last := time.Now()
	acc := 0.0
	var m msgT
	for {
		frameStart := time.Now()
		for {
			r, _, _ := pPeekMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1)
			if r == 0 {
				break
			}
			if m.Message == wmQuit {
				return
			}
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
		}
		if secondShut {
			secondShut = false
			g.Set.Windows = 1
			g.Set.save()
		}
		pollKeys()

		now := time.Now()
		dt := now.Sub(last).Seconds()
		last = now
		if dt > 0.1 {
			dt = 0.1
		}
		acc += dt
		for acc >= step {
			g.Update(step, &keys)
			acc -= step
		}
		if g.Quit {
			g.Quit = false
			if secondWnd != 0 {
				pDestroyWindow.Call(secondWnd)
				secondWnd = 0
			}
			pDestroyWindow.Call(mainWnd)
			continue
		}

		// open or close the second window as the game asks
		if n := g.Screens(); n == 2 && secondWnd == 0 {
			secondWnd = openWindow("Combine-Baler - second screen", secondArea())
		} else if n == 1 && secondWnd != 0 {
			pDestroyWindow.Call(secondWnd)
			secondWnd = 0
		}
		blit(mainWnd, g.RenderScreen(0))
		if secondWnd != 0 {
			blit(secondWnd, g.RenderScreen(1))
		}

		if d := time.Second/60 - time.Since(frameStart); d > 0 {
			time.Sleep(d)
		}
	}
}
