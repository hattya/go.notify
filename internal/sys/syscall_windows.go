//
// go.notify/internal/sys :: syscall_windows.go
//
//   Copyright (c) 2017-2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output z$GOFILE $GOFILE

package sys

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_WIN32_WINNT_WINXP = 0x0501
	_WIN32_WINNT_WIN7  = 0x0601
)

type OSVersionInfoEx struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformId        uint32
	CSDVersion        [128]uint16
	ServicePackMajor  uint16
	ServicePackMinor  uint16
	SuiteMask         uint16
	ProductType       uint8
	Reserved          uint8
}

const (
	VER_MINORVERSION = 1 << iota
	VER_MAJORVERSION
	VER_BUILDNUMBER
	VER_PLATFORMID
	VER_SERVICEPACKMINOR
	VER_SERVICEPACKMAJOR
	VER_SUITENAME
	VER_PRODUCT_TYPE
)

const (
	VER_EQUAL = 1 + iota
	VER_GREATER
	VER_GREATER_EQUAL
	VER_LESS
	VER_LESS_EQUAL
	VER_AND
	VER_OR
)

func IsWindowsXPSP2OrGreater() bool {
	return IsWindowsVersionOrGreater(_WIN32_WINNT_WINXP>>8&0xff, _WIN32_WINNT_WINXP&0xff, 2)
}

func IsWindows7OrGreater() bool {
	return IsWindowsVersionOrGreater(_WIN32_WINNT_WIN7>>8&0xff, _WIN32_WINNT_WIN7&0xff, 0)
}

func IsWindowsVersionOrGreater(major, minor uint32, sp uint16) bool {
	vi := OSVersionInfoEx{
		MajorVersion:     major,
		MinorVersion:     minor,
		ServicePackMajor: sp,
	}
	vi.OSVersionInfoSize = uint32(unsafe.Sizeof(vi))
	return VerifyVersionInfo(&vi, VER_MAJORVERSION|VER_MINORVERSION|VER_SERVICEPACKMAJOR, VerSetConditionMask(VerSetConditionMask(VerSetConditionMask(0, VER_MAJORVERSION, VER_GREATER_EQUAL), VER_MINORVERSION, VER_GREATER_EQUAL), VER_SERVICEPACKMAJOR, VER_GREATER_EQUAL))
}

func LoWord(v uint32) uint16 {
	return uint16(v & 0xffff)
}

//sys	GetModuleHandle(name *uint16) (h windows.Handle, err error) = GetModuleHandleW
//sys	VerifyVersionInfo(vi *OSVersionInfoEx, typeMask uint32, conditionMask uint64) (ok bool) = VerifyVersionInfoW
//sys	VerSetConditionMask(lConditionMask uint64, typeBitMask uint32, conditionMask uint8) (mask uint64)

const GWL_USERDATA = -21

const (
	IMAGE_BITMAP = iota
	IMAGE_ICON
	IMAGE_CURSOR
)

const (
	LR_DEFAULTSIZE = 0x00000040
	LR_SHARED      = 0x00008000
)

const OIC_SAMPLE = 32512

const (
	MF_STRING    = 0x00000000
	MF_POPUP     = 0x00000010
	MF_SEPARATOR = 0x00000800
)

const TPM_RIGHTALIGN = 0x0008

const (
	WM_CLOSE       = 0x0010
	WM_COMMAND     = 0x0111
	WM_CONTEXTMENU = 0x007b
	WM_CREATE      = 0x0001
	WM_DESTROY     = 0x0002
	WM_NULL        = 0x0000
	WM_RBUTTONUP   = 0x0205
	WM_SYSKEYDOWN  = 0x0104
	WM_USER        = 0x0400
)

const WS_POPUP = 0x80000000

type CreateStruct struct {
	CreateParams unsafe.Pointer
	Instance     windows.Handle
	Menu         windows.Handle
	Parent       windows.Handle
	Cy           int32
	Cx           int32
	Y            int32
	X            int32
	Style        int32
	Name         *uint16
	Class        *uint16
	ExStyle      uint32
}

type IconInfo struct {
	Icon     int32
	XHotspot uint32
	YHotspot uint32
	Mask     windows.Handle
	Color    windows.Handle
}

type Msg struct {
	Wnd     windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      Point
}

type Point struct {
	X, Y int32
}

type WndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

//go:nocheckptr
func MakeIntResource(i uint16) *uint16 {
	return (*uint16)(unsafe.Pointer(uintptr(i)))
}

//sys	AppendMenu(menu windows.Handle, flags uint32, id uintptr, item *uint16) (err error) = user32.AppendMenuW
//sys	CreateIconIndirect(ii *IconInfo) (icon windows.Handle, err error) = user32.CreateIconIndirect
//sys	CreatePopupMenu() (menu windows.Handle, err error) = user32.CreatePopupMenu
//sys	CreateWindowEx(exStyle uint32, className *uint16, windowName *uint16, style uint32, x int32, y int32, w int32, h int32, parent windows.Handle, menu windows.Handle, inst windows.Handle, param unsafe.Pointer) (wnd windows.Handle, err error) = user32.CreateWindowExW
//sys	DefWindowProc(wnd windows.Handle, msg uint32, wParam uintptr, lParam uintptr) (res uintptr) = user32.DefWindowProcW
//sys	DestroyIcon(icon windows.Handle) (err error) = user32.DestroyIcon
//sys	DestroyMenu(menu windows.Handle) (err error) = user32.DestroyMenu
//sys	DestroyWindow(wnd windows.Handle) (err error) = user32.DestroyWindow
//sys	DispatchMessage(msg *Msg) (res uintptr) = user32.DispatchMessageW
//sys	GetCursorPos(pt *Point) (err error) = user32.GetCursorPos
//sys	GetDC(wnd windows.Handle) (dc windows.Handle, err error) = user32.GetDC
//sys	GetMessage(msg *Msg, wnd windows.Handle, msgFilterMin uint32, msgFilterMax uint32) (ret int32, err error) [failretval==-1] = user32.GetMessageW
//sys	getWindowLong(wnd windows.Handle, i int32) (ptr uintptr, err error) = user32.GetWindowLongW
//sys	getWindowLongPtr(wnd windows.Handle, i int32) (ptr uintptr, err error) = user32.GetWindowLongPtrW
//sys	LoadImage(inst windows.Handle, name *uint16, typ uint32, cxDesired int32, cyDesired int32, load uint32) (h windows.Handle, err error) = user32.LoadImageW
//sys	PostMessage(wnd windows.Handle, msg uint32, wParam uintptr, lParam uintptr) (err error) = user32.PostMessageW
//sys	PostQuitMessage(exitCode int32) = user32.PostQuitMessage
//sys	RegisterClassEx(wcx *WndClassEx) (atom uint16, err error) = user32.RegisterClassExW
//sys	RegisterWindowMessage(s *uint16) (msg uint32, err error) = user32.RegisterWindowMessageW
//sys	ReleaseDC(wnd windows.Handle, dc windows.Handle) (err error) = user32.ReleaseDC
//sys	SetForegroundWindow(wnd windows.Handle) (err error) = user32.SetForegroundWindow
//sys	setWindowLong(wnd windows.Handle, i int32, ptr unsafe.Pointer) (oldptr uintptr, err error) = user32.SetWindowLongW
//sys	setWindowLongPtr(wnd windows.Handle, i int32, ptr unsafe.Pointer) (oldptr uintptr, err error) = user32.SetWindowLongPtrW
//sys	TrackPopupMenu(menu windows.Handle, flags uint32, x int32, y int32, reserved int32, wnd windows.Handle) (ret int32, err error) = user32.TrackPopupMenu
//sys	TranslateMessage(msg *Msg) (err error) = user32.TranslateMessage

func RGB(r, g, b uint8) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

//sys	CreateCompatibleBitmap(dc windows.Handle, w int32, h int32) (bm windows.Handle, err error) = gdi32.CreateCompatibleBitmap
//sys	CreateCompatibleDC(dc windows.Handle) (mdc windows.Handle, err error) = gdi32.CreateCompatibleDC
//sys	DeleteDC(dc windows.Handle) (err error) = gdi32.DeleteDC
//sys	DeleteObject(obj windows.Handle) (err error) = gdi32.DeleteObject
//sys	SelectObject(dc windows.Handle, obj windows.Handle) (oldobj windows.Handle, err error) = gdi32.SelectObject
//sys	SetPixel(dc windows.Handle, x int32, y int32, color uint32) (err error) [failretval==^uintptr(0)] = gdi32.SetPixel

type DLLVersionInfo struct {
	Size         uint32
	MajorVersion uint32
	MinorVersion uint32
	BuildNumber  uint32
	PlatformID   uint32
}

const (
	DLLVER_PLATFORM_WINDOWS = 1 + iota
	DLLVER_PLATFORM_NT
)

func IsShellDLLVersionOrGreater(major, minor, build uint32) bool {
	var dvi DLLVersionInfo
	pDllGetVersion := modshell32.NewProc("DllGetVersion")
	if pDllGetVersion.Find() != nil {
		dvi.MajorVersion = 4
	} else {
		dvi.Size = uint32(unsafe.Sizeof(dvi))
		r0, _, _ := pDllGetVersion.Call(uintptr(unsafe.Pointer(&dvi)))
		if r0 != 0 {
			return false
		}
	}
	return dvi.MajorVersion<<16|dvi.MinorVersion<<8|dvi.BuildNumber >= major<<16|minor<<8|build
}

const (
	NIM_ADD = iota
	NIM_MODIFY
	NIM_DELETE
	NIM_SETFOCUS
	NIM_SETVERSION
)

const (
	NOTIFYICONDATA_V3_SIZE = uint32(unsafe.Offsetof(NotifyIconData{}.BalloonIcon))
	NOTIFYICONDATA_V2_SIZE = uint32(unsafe.Offsetof(NotifyIconData{}.GuidItem))
	NOTIFYICONDATA_V1_SIZE = uint32(unsafe.Offsetof(NotifyIconData{}.Tip) + unsafe.Sizeof([64]uint16{}))
)

type NotifyIconData struct {
	Size            uint32
	Wnd             windows.Handle
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            windows.Handle
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GuidItem        windows.GUID
	BalloonIcon     windows.Handle
}

const (
	NIF_MESSAGE = 1 << iota
	NIF_ICON
	NIF_TIP
	NIF_STATE
	NIF_INFO
	NIF_GUID
	NIF_REALTIME
	NIF_SHOWTIP
)

const (
	NIN_BALLOONSHOW = WM_USER + 2 + iota
	NIN_BALLOONHIDE
	NIN_BALLOONTIMEOUT
	NIN_BALLOONUSERCLICK
)

const (
	NIS_HIDDEN = 1 << iota
	NIS_SHAREDICON
)

const (
	NOTIFY_VERSION   = 3
	NOTIFY_VERSION_4 = 4
)

const (
	NIIF_NONE = iota
	NIIF_INFO
	NIIF_WARNING
	NIIF_ERROR
	NIIF_USER
	NIIF_NOSOUND = 1 << (iota - 1)
	NIIF_LARGE_ICON
	_
	NIIF_RESPECT_QUIET_TIME

	NIIF_ICON_MASK = 0xf
)

//sys	Shell_NotifyIcon(message uint32, data *NotifyIconData) (err error) = shell32.Shell_NotifyIconW
