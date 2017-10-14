//
// go.notify/windows :: notify_windows.go
//
//   Copyright (c) 2017 Akinori Hattori <hattya@gmail.com>
//
//   Permission is hereby granted, free of charge, to any person
//   obtaining a copy of this software and associated documentation files
//   (the "Software"), to deal in the Software without restriction,
//   including without limitation the rights to use, copy, modify, merge,
//   publish, distribute, sublicense, and/or sell copies of the Software,
//   and to permit persons to whom the Software is furnished to do so,
//   subject to the following conditions:
//
//   The above copyright notice and this permission notice shall be
//   included in all copies or substantial portions of the Software.
//
//   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
//   EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
//   MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
//   NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
//   BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
//   ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
//   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//   SOFTWARE.
//

//go:generate stringer -type IconType -output notify_string_windows.go

// Package windows implements the Windows Notifications.
//
// See https://msdn.microsoft.com/en-us/library/windows/desktop/ee330740(v=vs.85).aspx
// for details.
package windows

import (
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/hattya/go.notify/internal/sys"
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
var testHookPrepare func(*NotifyIcon)

// NotifyIcon represents a notification icon in the notification area.
type NotifyIcon struct {
	GUID GUID // requires Windows 7 or later

	name string
	wnd  windows.Handle
	wg   sync.WaitGroup
	err  chan error

	mu    sync.Mutex
	data  sys.NotifyIconData
	added int32
	done  int32
}

// New returns a new NotifyIcon.
func New(name string) (ni *NotifyIcon, err error) {
	ni = &NotifyIcon{
		name: name,
		err:  make(chan error, 1),
	}
	// shell32.dll version
	switch {
	case sys.IsShellDLLVersionOrGreater(6, 0, 6):
		ni.data.Size = uint32(unsafe.Sizeof(ni.data))
		ni.data.Flags |= sys.NIF_SHOWTIP
		ni.data.Version = sys.NOTIFY_VERSION_4
		ni.data.InfoFlags |= sys.NIIF_LARGE_ICON
		if sys.IsWindows7OrGreater() {
			ni.data.InfoFlags |= sys.NIIF_RESPECT_QUIET_TIME
		}
	case sys.IsShellDLLVersionOrGreater(6, 0, 0):
		ni.data.Size = sys.NOTIFYICONDATA_V3_SIZE
		ni.data.Version = sys.NOTIFY_VERSION
	case sys.IsShellDLLVersionOrGreater(5, 0, 0):
		ni.data.Size = sys.NOTIFYICONDATA_V2_SIZE
		ni.data.Version = sys.NOTIFY_VERSION
	default:
		ni.data.Size = sys.NOTIFYICONDATA_V1_SIZE
	}
	// tooltip
	ni.data.Flags |= sys.NIF_TIP
	u, err := windows.UTF16FromString(ni.name)
	if err != nil {
		return
	}
	copy(ni.data.Tip[:], u)

	ni.wg.Add(1)
	go ni.loop()
	if err = <-ni.err; err != nil {
		ni = nil
	}
	return
}

// Close deletes the NotifyIcon from the notification area.
func (ni *NotifyIcon) Close() error {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	if atomic.LoadInt32(&ni.done) == 1 {
		return nil
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
	case ni.GUID != "":
		if !sys.IsWindows7OrGreater() {
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
	default:
		return ErrIcon
	}
	// sound
	if !n.Sound {
		if !sys.IsShellDLLVersionOrGreater(6, 0, 0) {
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

func (ni *NotifyIcon) loop() {
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
		atomic.StoreInt32(&ni.done, 1)
		sys.PostQuitMessage(0)
		ni.err <- err
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
)

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

// VersionError represents that it requires newer Windows version to perform
// the specified operation.
type VersionError string

func (e VersionError) Error() string {
	return fmt.Sprintf("go.notify: require Windows %v or later", string(e))
}
