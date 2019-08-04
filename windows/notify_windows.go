//
// go.notify/windows :: notify_windows.go
//
//   Copyright (c) 2017-2019 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

//go:generate stringer -type BalloonEvent,IconType -output notify_string_windows.go

// Package windows implements the Windows Notifications.
//
// See https://msdn.microsoft.com/en-us/library/windows/desktop/ee330740(v=vs.85).aspx
// for details.
package windows

import (
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/hattya/go.notify/internal/sys"
	"github.com/hattya/go.notify/internal/util"
	"golang.org/x/sys/windows"
)

var (
	ErrGUID = errors.New("go.notify: invalid GUID format")
	ErrIcon = errors.New("go.notify: unknown icon type")
)

const className = "go.notify.Window"

var _WM_TASKBARCREATED uint32

func init() {
	p, _ := windows.UTF16PtrFromString(className)
	wcx := &sys.WndClassEx{
		WndProc:   windows.NewCallback(windowProc),
		ClassName: p,
	}
	wcx.Size = uint32(unsafe.Sizeof(*wcx))
	if _, err := sys.RegisterClassEx(wcx); err != nil {
		panic(err)
	}

	p, _ = windows.UTF16PtrFromString("TaskbarCreated")
	wm, err := sys.RegisterWindowMessage(p)
	if err != nil {
		panic(err)
	}
	_WM_TASKBARCREATED = wm
}

// for testing
var (
	isShellDLLVersionOrGreater = sys.IsShellDLLVersionOrGreater
	isWindows7OrGreater        = sys.IsWindows7OrGreater
	isWindowsXPSP2OrGreater    = sys.IsWindowsXPSP2OrGreater
	loadImage                  = sys.LoadImage
	testHookPrepare            func(*NotifyIcon)
	testHookNotify             func(*Notification)
)

// NotifyIcon represents a notification icon in the notification area.
type NotifyIcon struct {
	Icon    *Icon
	GUID    GUID              // requires Windows 7 or later
	Balloon chan BalloonEvent // requires Windows XP or later
	Menu    chan MenuEvent

	name string
	wnd  windows.Handle
	menu *Menu
	wg   sync.WaitGroup
	err  chan error

	mu    sync.Mutex
	data  sys.NotifyIconData
	added int32
	ev    chan interface{}
	done  chan struct{}
}

// New returns a new NotifyIcon.
func New(name string) (ni *NotifyIcon, err error) {
	ni = &NotifyIcon{
		Balloon: make(chan BalloonEvent),
		Menu:    make(chan MenuEvent),
		name:    name,
		err:     make(chan error, 1),
		ev:      make(chan interface{}),
		done:    make(chan struct{}),
	}
	// shell32.dll version
	switch {
	case isShellDLLVersionOrGreater(6, 0, 6):
		ni.data.Size = uint32(unsafe.Sizeof(ni.data))
		ni.data.Flags |= sys.NIF_SHOWTIP
		ni.data.Version = sys.NOTIFY_VERSION_4
		ni.data.InfoFlags |= sys.NIIF_LARGE_ICON
		if isWindows7OrGreater() {
			ni.data.InfoFlags |= sys.NIIF_RESPECT_QUIET_TIME
		}
	case isShellDLLVersionOrGreater(6, 0, 0):
		ni.data.Size = sys.NOTIFYICONDATA_V3_SIZE
		ni.data.Version = sys.NOTIFY_VERSION
	case isShellDLLVersionOrGreater(5, 0, 0):
		ni.data.Size = sys.NOTIFYICONDATA_V2_SIZE
		ni.data.Version = sys.NOTIFY_VERSION
	default:
		ni.data.Size = sys.NOTIFYICONDATA_V1_SIZE
	}
	// message
	ni.data.Flags |= sys.NIF_MESSAGE
	ni.data.CallbackMessage = sys.WM_USER
	// tooltip
	ni.data.Flags |= sys.NIF_TIP
	u, err := windows.UTF16FromString(ni.name)
	if err != nil {
		return
	}
	copy(ni.data.Tip[:], u)

	ni.wg.Add(2)
	go ni.event()
	go ni.message()
	if err = <-ni.err; err != nil {
		ni = nil
	}
	return
}

// Close deletes the NotifyIcon from the notification area.
func (ni *NotifyIcon) Close() error {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	select {
	case <-ni.done:
		return nil
	default:
	}

	sys.PostMessage(ni.wnd, sys.WM_CLOSE, 0, 0)
	ni.wg.Wait()
	return <-ni.err
}

// Add adds the NotifyIcon to the notification area.
func (ni *NotifyIcon) Add() error {
	ni.mu.Lock()
	defer ni.mu.Unlock()

	if err := ni.prepare(); err != nil {
		return err
	}
	return ni.add(&ni.data)
}

// Modify modifies the NotifyIcon in the notification area.
func (ni *NotifyIcon) Modify() error {
	ni.mu.Lock()
	defer ni.mu.Unlock()

	if err := ni.prepare(); err != nil {
		return err
	}
	return sys.Shell_NotifyIcon(sys.NIM_MODIFY, &ni.data)
}

func (ni *NotifyIcon) prepare() error {
	switch {
	case ni.Icon != nil:
		ni.data.Flags |= sys.NIF_ICON
		ni.data.Icon = ni.Icon.h
	case ni.data.Flags&sys.NIF_ICON != 0:
		ni.data.Flags ^= sys.NIF_ICON
		ni.data.Icon = 0
	}

	switch {
	case ni.GUID != "":
		if !isWindows7OrGreater() {
			return VersionError("7")
		}
		guid, err := ni.GUID.parse()
		if err != nil {
			return err
		}
		ni.data.Flags |= sys.NIF_GUID
		ni.data.GuidItem = guid
	case ni.data.Flags&sys.NIF_GUID != 0:
		ni.data.Flags ^= sys.NIF_GUID
		ni.data.GuidItem = windows.GUID{}
	}

	if testHookPrepare != nil {
		testHookPrepare(ni)
	}
	return nil
}

// Notify displays a notification.
func (ni *NotifyIcon) Notify(n *Notification) error {
	// copy
	ni.mu.Lock()
	if err := ni.prepare(); err != nil {
		ni.mu.Unlock()
		return err
	}
	data := ni.data
	ni.mu.Unlock()

	if testHookNotify != nil {
		testHookNotify(n)
	}

	data.Flags |= sys.NIF_INFO
	u, err := windows.UTF16FromString(n.Title)
	if err != nil {
		return err
	}
	copy(data.InfoTitle[:], u)
	u, err = windows.UTF16FromString(n.Body)
	if err != nil {
		return err
	}
	copy(data.Info[:], u)
	// icon
	switch n.IconType {
	case IconNone, IconInfo, IconWarn, IconError:
		data.InfoFlags |= uint32(n.IconType)
	case IconUser:
		if !isWindowsXPSP2OrGreater() {
			return VersionError("XP SP2")
		}
		data.InfoFlags |= uint32(n.IconType)
		if n.Icon != nil {
			if !isShellDLLVersionOrGreater(6, 0, 6) {
				return VersionError("Vista")
			}
			data.BalloonIcon = n.Icon.h
		}
	default:
		return ErrIcon
	}
	// sound
	if !n.Sound {
		if !isShellDLLVersionOrGreater(6, 0, 0) {
			return VersionError("XP")
		}
		data.InfoFlags |= sys.NIIF_NOSOUND
	}

	if atomic.LoadInt32(&ni.added) == 0 {
		return ni.add(&data)
	}
	return sys.Shell_NotifyIcon(sys.NIM_MODIFY, &data)
}

func (ni *NotifyIcon) add(data *sys.NotifyIconData) error {
	err := sys.Shell_NotifyIcon(sys.NIM_ADD, data)
	if err == nil {
		if data.Version != 0 {
			sys.Shell_NotifyIcon(sys.NIM_SETVERSION, data)
		}
		atomic.StoreInt32(&ni.added, 1)
	}
	return err
}

// CreateMenu creates a new context menu.
func (ni *NotifyIcon) CreateMenu() *Menu {
	ni.menu = new(Menu)
	return ni.menu
}

func (ni *NotifyIcon) event() {
	defer ni.wg.Done()

	var balloon chan BalloonEvent
	var menu chan MenuEvent
	balloonBuf := make([]BalloonEvent, 1)
	menuBuf := make([]MenuEvent, 1)

	for {
		select {
		case ev := <-ni.ev:
			switch ev := ev.(type) {
			case BalloonEvent:
				if balloon == nil {
					balloon = ni.Balloon
					balloonBuf = balloonBuf[1:]
				}
				balloonBuf = append(balloonBuf, ev)
			case MenuEvent:
				if menu == nil {
					menu = ni.Menu
					menuBuf = menuBuf[1:]
				}
				menuBuf = append(menuBuf, ev)
			}
		case balloon <- balloonBuf[0]:
			if len(balloonBuf) == 1 {
				balloon = nil
			} else {
				balloonBuf = balloonBuf[1:]
			}
		case menu <- menuBuf[0]:
			if len(menuBuf) == 1 {
				menu = nil
			} else {
				menuBuf = menuBuf[1:]
			}
		case <-ni.done:
			return
		}
	}
}

func (ni *NotifyIcon) message() {
	defer ni.wg.Done()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cp, _ := windows.UTF16PtrFromString(className)
	wp, _ := windows.UTF16PtrFromString(ni.name)
	// create WS_POPUP window to receive TaskbarCreated message
	wnd, err := sys.CreateWindowEx(0, cp, wp, sys.WS_POPUP, 0, 0, 0, 0, 0, 0, 0, unsafe.Pointer(ni))
	if err != nil {
		ni.err <- err
		return
	}
	ni.wnd = wnd
	ni.data.Wnd = wnd
	ni.err <- nil

	var msg sys.Msg
	for {
		rv, _ := sys.GetMessage(&msg, 0, 0, 0)
		if rv <= 0 {
			break
		}
		sys.TranslateMessage(&msg)
		sys.DispatchMessage(&msg)
	}
}

func (ni *NotifyIcon) windowProc(wnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case sys.WM_CLOSE:
		sys.DestroyWindow(wnd)
	case sys.WM_DESTROY:
		var err error
		if atomic.LoadInt32(&ni.added) != 0 {
			err = sys.Shell_NotifyIcon(sys.NIM_DELETE, &ni.data)
		}
		close(ni.done)
		sys.PostQuitMessage(0)
		ni.err <- err
	case sys.WM_USER:
		switch sys.LoWord(uint32(lParam)) {
		case sys.WM_RBUTTONUP:
			sys.PostMessage(wnd, sys.WM_CONTEXTMENU, 0, 0)
		case sys.NIN_BALLOONSHOW:
			ni.ev <- BalloonShown
		case sys.NIN_BALLOONHIDE:
			ni.ev <- BalloonClosed
		case sys.NIN_BALLOONTIMEOUT:
			ni.ev <- BalloonClosed
		case sys.NIN_BALLOONUSERCLICK:
			ni.ev <- BalloonClicked
		}
	case sys.WM_CONTEXTMENU:
		var pt sys.Point
		sys.GetCursorPos(&pt)
		sys.SetForegroundWindow(wnd)
		menu, err := ni.menu.sys()
		if err != nil {
			panic(err)
		}
		sys.TrackPopupMenu(menu, sys.TPM_RIGHTALIGN, pt.X, pt.Y, 0, wnd)
		sys.DestroyMenu(menu)
		sys.PostMessage(wnd, sys.WM_NULL, 0, 0)
	case sys.WM_COMMAND:
		ni.ev <- MenuEvent{ID: sys.LoWord(uint32(wParam))}
	case sys.WM_SYSKEYDOWN:
		// disable Alt+F4
	default:
		if msg == _WM_TASKBARCREATED {
			atomic.StoreInt32(&ni.added, 0)
			if err := ni.Add(); err != nil {
				panic(err)
			}
		}
		return sys.DefWindowProc(wnd, msg, wParam, lParam)
	}
	return 0
}

func windowProc(wnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == sys.WM_CREATE {
		cs := (*sys.CreateStruct)(unsafe.Pointer(lParam))
		sys.SetWindowLongPtr(wnd, sys.GWL_USERDATA, cs.CreateParams)
		return 0
	} else if ptr, err := sys.GetWindowLongPtr(wnd, sys.GWL_USERDATA); err == nil {
		ni := (*NotifyIcon)(unsafe.Pointer(ptr))
		return ni.windowProc(wnd, msg, wParam, lParam)
	}
	return sys.DefWindowProc(wnd, msg, wParam, lParam)
}

// Notification represents a notification.
type Notification struct {
	Title    string
	Body     string
	IconType IconType
	Icon     *Icon // requires Windows Vista or later
	Sound    bool
}

// IconType represents an icon type of the Notification.
type IconType uint8

// List of icon types for the Notification.
const (
	IconNone  IconType = sys.NIIF_NONE
	IconInfo  IconType = sys.NIIF_INFO
	IconWarn  IconType = sys.NIIF_WARNING
	IconError IconType = sys.NIIF_ERROR
	IconUser  IconType = sys.NIIF_USER // requires Windows XP SP2 or later
)

// Icon represents a handle of the icon.
type Icon struct {
	h windows.Handle
}

// LoadImage returns a new Icon from the specified img.
func LoadImage(img image.Image) (icon *Icon, err error) {
	img, err = util.Convert(img)
	if err != nil {
		return
	}
	size := img.Bounds().Size()

	dc, err := sys.GetDC(0)
	if err != nil {
		return
	}
	defer sys.ReleaseDC(0, dc)
	// bitmask bitmap
	mdc1, err := sys.CreateCompatibleDC(dc)
	if err != nil {
		return
	}
	defer sys.DeleteDC(mdc1)
	mask, err := sys.CreateCompatibleBitmap(dc, int32(size.X), int32(size.Y))
	if err != nil {
		return
	}
	defer sys.DeleteObject(mask)
	obj1, err := sys.SelectObject(mdc1, mask)
	if err != nil {
		return
	}
	defer sys.SelectObject(mdc1, obj1)
	// color bitmap
	mdc2, err := sys.CreateCompatibleDC(dc)
	if err != nil {
		return
	}
	defer sys.DeleteDC(mdc2)
	bm, err := sys.CreateCompatibleBitmap(dc, int32(size.X), int32(size.Y))
	if err != nil {
		return
	}
	defer sys.DeleteObject(bm)
	obj2, err := sys.SelectObject(mdc2, bm)
	if err != nil {
		return
	}
	defer sys.SelectObject(mdc2, obj2)

	switch img := img.(type) {
	case *image.Gray:
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				c := img.GrayAt(x, y)
				sys.SetPixel(mdc2, int32(x), int32(y), sys.RGB(c.Y, c.Y, c.Y))
			}
		}
	case *image.NRGBA:
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				c := img.NRGBAAt(x, y)
				a := 255 - c.A
				sys.SetPixel(mdc1, int32(x), int32(y), sys.RGB(a, a, a))
				sys.SetPixel(mdc2, int32(x), int32(y), sys.RGB(c.R, c.G, c.B))
			}
		}
	}
	// deselect from device context
	sys.SelectObject(mdc1, obj1)
	sys.SelectObject(mdc2, obj2)
	h, err := sys.CreateIconIndirect(&sys.IconInfo{
		Icon:     1,
		XHotspot: 0,
		YHotspot: 0,
		Mask:     mask,
		Color:    bm,
	})
	if err == nil {
		icon = &Icon{h: h}
		runtime.SetFinalizer(icon, (*Icon).Close)
	}
	return
}

// LoadIcon returns a new Icon from the specified icon resource.
func LoadIcon(i uint16) (icon *Icon, err error) {
	inst, err := sys.GetModuleHandle(nil)
	if err != nil {
		return
	}
	h, err := loadImage(inst, sys.MakeIntResource(i), sys.IMAGE_ICON, 0, 0, sys.LR_DEFAULTSIZE)
	if err == nil {
		icon = &Icon{h: h}
	}
	return
}

// Close destroys the Icon.
func (icon *Icon) Close() error {
	err := sys.DestroyIcon(icon.h)
	runtime.SetFinalizer(icon, nil)
	return err
}

// GUID represents a textual representation of the GUID.
//
// The following formats are supported ("x" represents a hexadecimal digit):
//   - xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
//   - xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
//   - {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}
type GUID string

func (g GUID) parse() (guid windows.GUID, err error) {
	s := string(g)
	var u uint64
	var b []byte
	switch len(s) {
	case 0:
		goto Error
	case 32:
		if u, err = strconv.ParseUint(s[:8], 16, 32); err != nil {
			goto Error
		}
		guid.Data1 = uint32(u)
		if u, err = strconv.ParseUint(s[8:12], 16, 16); err != nil {
			goto Error
		}
		guid.Data2 = uint16(u)
		if u, err = strconv.ParseUint(s[12:16], 16, 16); err != nil {
			goto Error
		}
		guid.Data3 = uint16(u)
		if b, err = hex.DecodeString(s[16:]); err != nil {
			goto Error
		}
		copy(guid.Data4[:], b)
	default:
		if s[0] == '{' {
			if s[len(s)-1] != '}' {
				goto Error
			}
			s = s[1 : len(s)-1]
		}
		for i, n := range []int{8, 4, 4, 4, 12} {
			if 0 < i {
				if s == "" || s[0] != '-' {
					goto Error
				}
				s = s[1:]
			}
			if len(s) < n {
				goto Error
			}
			switch i {
			case 0:
				u, err = strconv.ParseUint(s[:n], 16, 32)
				guid.Data1 = uint32(u)
			case 1:
				u, err = strconv.ParseUint(s[:n], 16, 16)
				guid.Data2 = uint16(u)
			case 2:
				u, err = strconv.ParseUint(s[:n], 16, 16)
				guid.Data3 = uint16(u)
			case 3:
				b, err = hex.DecodeString(s[:n])
				copy(guid.Data4[:], b)
			case 4:
				b, err = hex.DecodeString(s[:n])
				copy(guid.Data4[2:], b)
			}
			if err != nil {
				goto Error
			}
			s = s[n:]
		}
	}
	return
Error:
	return windows.GUID{}, ErrGUID
}

// BalloonEvent represents an event of the notification balloon.
type BalloonEvent uint

// List of events for the notification balloon.
const (
	// BalloonShown represents the NIN_BALLOONSHOW message.
	BalloonShown BalloonEvent = iota

	// BalloonClosed represents the NIN_BALLOONHIDE or NIN_BALLOONTIMEOUT message.
	BalloonClosed

	// BalloonClicked represents the NIN_BALLOONUSERCLICK message.
	BalloonClicked
)

// Menu represents a context menu of the NotifyIcon.
type Menu struct {
	items []menuItem
}

// Item appends an item to the context menu.
func (m *Menu) Item(text string, id uint) {
	m.items = append(m.items, menuItem{
		text:  text,
		id:    id,
		flags: sys.MF_STRING,
	})
}

// Submenu appends a submenu to the context menu.
func (m *Menu) Submenu(text string) *Menu {
	menu := new(Menu)
	m.items = append(m.items, menuItem{
		text:  text,
		flags: sys.MF_POPUP,
		menu:  menu,
	})
	return menu
}

// Sep appends a separator to the context menu.
func (m *Menu) Sep() {
	m.items = append(m.items, menuItem{
		flags: sys.MF_SEPARATOR,
	})
}

func (m *Menu) sys() (windows.Handle, error) {
	menu, err := sys.CreatePopupMenu()
	if err != nil {
		return 0, err
	}
	for _, mi := range m.items {
		var item uintptr
		if mi.menu != nil {
			sub, err := mi.menu.sys()
			if err != nil {
				sys.DestroyMenu(menu)
				return 0, err
			}
			item = uintptr(sub)
		} else {
			item = uintptr(mi.id)
		}
		p, err := windows.UTF16PtrFromString(mi.text)
		if err != nil {
			sys.DestroyMenu(menu)
			return 0, err
		}
		if err := sys.AppendMenu(menu, mi.flags, item, p); err != nil {
			sys.DestroyMenu(menu)
			return 0, err
		}
	}
	return menu, nil
}

type menuItem struct {
	text  string
	id    uint
	flags uint32
	menu  *Menu
}

// MenuEvent represents an event of the context menu.
type MenuEvent struct {
	ID uint16
}

// VersionError represents that it requires newer Windows version to perform
// the specified operation.
type VersionError string

func (e VersionError) Error() string {
	return fmt.Sprintf("go.notify: require Windows %v or later", string(e))
}
