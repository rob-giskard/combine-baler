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
	wmQuit       = 0x0012
	wmEraseBkgnd = 0x0014
	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
)

var keys [256]bool

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	case wmKeyDown:
		keys[wp&0xFF] = true
		return 0
	case wmKeyUp:
		keys[wp&0xFF] = false
		return 0
	case wmKillFocus:
		keys = [256]bool{}
	case wmEraseBkgnd:
		return 1
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
	return r
}

func init() { runtime.LockOSThread() }

func fatal(s string) {
	os.WriteFile("combine_baler_crash.txt", []byte(s), 0644)
	t, _ := syscall.UTF16PtrFromString("Combine-Baler")
	m, _ := syscall.UTF16PtrFromString(s)
	pMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x10)
	os.Exit(1)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fatal(fmt.Sprintf("Combine-Baler crashed: %v\n\n%s", r, debug.Stack()))
		}
	}()
	pSetDPIAware.Call()
	pTimeBeginPeriod.Call(1)

	hInst, _, _ := pGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("CombineBalerWindow")
	title, _ := syscall.UTF16PtrFromString("Combine-Baler")
	cursor, _, _ := pLoadCursor.Call(0, 32512)

	wc := wndClassEx{
		Style:     0x0023, // CS_HREDRAW | CS_VREDRAW | CS_OWNDC
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
	scale := 1
	for (scale+1)*SW <= int(scrW)*9/10 && (scale+1)*SH <= int(scrH)*85/100 {
		scale++
	}
	const style = 0x00CF0000 // WS_OVERLAPPEDWINDOW
	rc := rectT{0, 0, int32(SW * scale), int32(SH * scale)}
	pAdjustWindowRect.Call(uintptr(unsafe.Pointer(&rc)), style, 0, 0)
	ww, wh := int(rc.Right-rc.Left), int(rc.Bottom-rc.Top)
	wx, wy := (int(scrW)-ww)/2, (int(scrH)-wh)/2
	if wx < 0 {
		wx = 0
	}
	if wy < 0 {
		wy = 0
	}
	hwnd, _, err := pCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		style|0x10000000, // | WS_VISIBLE
		uintptr(wx), uintptr(wy), uintptr(ww), uintptr(wh), 0, 0, hInst, 0)
	if hwnd == 0 {
		fatal(fmt.Sprintf("CreateWindowEx failed: %v", err))
	}
	hdc, _, _ := pGetDC.Call(hwnd)
	pSetStretchMode.Call(hdc, 3) // COLORONCOLOR: crisp pixels

	bmi := bitmapInfoHeader{Width: SW, Height: -SH, Planes: 1, BitCount: 32}
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
			pDestroyWindow.Call(hwnd)
			continue
		}
		g.Render()

		var cr rectT
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
		cw, ch := int(cr.Right), int(cr.Bottom)
		if cw > 0 && ch > 0 {
			dw, dh := cw, cw*SH/SW
			if dh > ch {
				dw, dh = ch*SW/SH, ch
			}
			if dw >= SW && dh >= SH { // snap to whole pixels when possible
				s := dw / SW
				if dh/SH < s {
					s = dh / SH
				}
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
				uintptr(unsafe.Pointer(&g.fb[0])), uintptr(unsafe.Pointer(&bmi)),
				0, 0x00CC0020) // DIB_RGB_COLORS, SRCCOPY
		}

		if d := time.Second/60 - time.Since(frameStart); d > 0 {
			time.Sleep(d)
		}
	}
}
