//
// go.notify/windows :: export_windows_test.go
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

package windows

import (
	"github.com/hattya/go.notify/internal/sys"
	"golang.org/x/sys/windows"
)

var (
	shellDLLVersion [][]uint32
	windowsVersion  []uint32
)

func MockShellDLLVersion(major, minor, build uint32) {
	shellDLLVersion = append(shellDLLVersion, []uint32{major, minor, build})
}

func MockWindows7() {
	windowsVersion = append(windowsVersion, 0x06010000)
}

func MockWindowsVista() {
	windowsVersion = append(windowsVersion, 0x06000000)
}

func MockWindows2000() {
	windowsVersion = append(windowsVersion, 0x0500000)
}

func init() {
	isShellDLLVersionOrGreater = func(major, minor, build uint32) bool {
		if len(shellDLLVersion) == 0 {
			return sys.IsShellDLLVersionOrGreater(major, minor, build)
		}
		v := shellDLLVersion[0]
		shellDLLVersion = shellDLLVersion[1:]
		return v[0]<<16|v[1]<<8|v[2] >= major<<16|minor<<8|build
	}
	isWindows7OrGreater = func() bool {
		if len(windowsVersion) == 0 {
			return sys.IsWindows7OrGreater()
		}
		v := windowsVersion[0]
		windowsVersion = windowsVersion[1:]
		return v >= 0x06010000
	}
	isWindowsXPSP2OrGreater = func() bool {
		if len(windowsVersion) == 0 {
			return sys.IsWindowsXPSP2OrGreater()
		}
		v := windowsVersion[0]
		windowsVersion = windowsVersion[1:]
		return v >= 0x05010200
	}
	loadImage = func(windows.Handle, *uint16, uint32, int32, int32, uint32) (windows.Handle, error) {
		return sys.LoadImage(0, sys.MakeIntResource(32512), sys.IMAGE_ICON, 0, 0, sys.LR_DEFAULTSIZE|sys.LR_SHARED)
	}
	testHookPrepare = func(ni *NotifyIcon) {
		if ni.data.Flags&sys.NIF_GUID != 0 {
			// test binary is in a temporary folder
			ni.data.Flags ^= sys.NIF_GUID
			ni.data.GuidItem = windows.GUID{}
		}
	}
	testHookNotify = func(n *Notification) {
		switch {
		case !sys.IsWindowsXPSP2OrGreater():
			if n.IconType == IconUser {
				n.IconType = IconInfo
				n.Icon = nil
			}
		case !sys.IsShellDLLVersionOrGreater(6, 0, 6):
			n.Icon = nil
		}
		if !sys.IsShellDLLVersionOrGreater(6, 0, 0) {
			n.Sound = true
		}
	}
}

func (ni *NotifyIcon) Data() sys.NotifyIconData {
	return ni.data
}

func (ni *NotifyIcon) Prepare(data sys.NotifyIconData) error {
	ni.data = data
	return ni.prepare()
}

func (ni *NotifyIcon) PostMessage(msg uint32, wParam, lParam uintptr) error {
	return sys.PostMessage(ni.wnd, msg, wParam, lParam)
}

func (g GUID) Parse() (windows.GUID, error) {
	return g.parse()
}

func (m *Menu) Sys() (windows.Handle, error) {
	return m.sys()
}
