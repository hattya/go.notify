//
// go.notify/windows :: notify_windows_test.go
//
//   Copyright (c) 2017-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package windows_test

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/hattya/go.notify/internal/sys"
	"github.com/hattya/go.notify/windows"
	syscall "golang.org/x/sys/windows"
)

const (
	name = "go.notify"
	guid = "{23977B55-10E0-4041-B862-B19541963669}"
)

func TestNew_Windows7(t *testing.T) {
	windows.MockShellDLLVersion(6, 0, 6)
	windows.MockWindows7()
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	data := ni.Data()
	if g, e := data.Size, uint32(unsafe.Sizeof(sys.NotifyIconData{})); g != e {
		t.Errorf("NotifyIconData.Size = %v, expected %v", g, e)
	}
	if data.InfoFlags&sys.NIIF_RESPECT_QUIET_TIME == 0 {
		t.Error("NIIF_RESPECT_QUIET_TIME is not set")
	}
}

func TestNew_WindowsVista(t *testing.T) {
	windows.MockShellDLLVersion(6, 0, 6)
	windows.MockWindowsVista()
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	data := ni.Data()
	if g, e := data.Size, uint32(unsafe.Sizeof(sys.NotifyIconData{})); g != e {
		t.Errorf("NotifyIconData.Size = %v, expected %v", g, e)
	}
	if data.InfoFlags&sys.NIIF_RESPECT_QUIET_TIME != 0 {
		t.Error("NIIF_RESPECT_QUIET_TIME is set")
	}
}

func TestNew_WindowsXP(t *testing.T) {
	windows.MockShellDLLVersion(6, 0, 0)
	windows.MockShellDLLVersion(6, 0, 0)
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	data := ni.Data()
	if g, e := data.Size, sys.NOTIFYICONDATA_V3_SIZE; g != e {
		t.Errorf("NotifyIconData.Size = %v, expected %v", g, e)
	}
}

func TestNew_Windows2000(t *testing.T) {
	windows.MockShellDLLVersion(5, 0, 0)
	windows.MockShellDLLVersion(5, 0, 0)
	windows.MockShellDLLVersion(5, 0, 0)
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	data := ni.Data()
	if g, e := data.Size, sys.NOTIFYICONDATA_V2_SIZE; g != e {
		t.Errorf("NotifyIconData.Size = %v, expected %v", g, e)
	}
}

func TestNew_Windows98(t *testing.T) {
	windows.MockShellDLLVersion(4, 72, 0)
	windows.MockShellDLLVersion(4, 72, 0)
	windows.MockShellDLLVersion(4, 72, 0)
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	data := ni.Data()
	if g, e := data.Size, sys.NOTIFYICONDATA_V1_SIZE; g != e {
		t.Errorf("NotifyIconData.Size = %v, expected %v", g, e)
	}
}

func TestNewError(t *testing.T) {
	if _, err := windows.New("\000"); err == nil {
		t.Error("expected error")
	}
}

func TestClose(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		if err := ni.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPrepare(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	icon, err := load()
	if err != nil {
		t.Error(err)
	}
	defer icon.Close()

	ni.Icon = icon
	if err := ni.Prepare(sys.NotifyIconData{}); err != nil {
		t.Fatal(err)
	}
	if data := ni.Data(); data.Flags&sys.NIF_ICON == 0 {
		t.Error("NIF_ICON is not set")
	}

	ni.Icon = nil
	if err := ni.Prepare(ni.Data()); err != nil {
		t.Fatal(err)
	}
	if data := ni.Data(); data.Flags&sys.NIF_ICON != 0 {
		t.Error("NIF_ICON is set")
	}

	windows.MockWindows7()
	ni.GUID = guid
	if err := ni.Prepare(sys.NotifyIconData{}); err != nil {
		t.Fatal(err)
	}

	windows.MockWindowsVista()
	ni.GUID = guid
	if err := ni.Prepare(sys.NotifyIconData{}); err == nil {
		t.Fatal("expected error")
	}

	ni.GUID = ""
	if err := ni.Prepare(sys.NotifyIconData{Flags: sys.NIF_GUID}); err != nil {
		t.Fatal(err)
	}
}

func TestAdd(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	icon, err := load()
	if err != nil {
		t.Error(err)
	}
	defer icon.Close()

	ni.Icon = icon
	if err := ni.Add(); err != nil {
		t.Error(err)
	}
	// already added
	if err := ni.Add(); err == nil {
		t.Error("expected error")
	}
	// invalid GUID
	windows.MockWindows7()
	ni.GUID = "{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}"
	if err := ni.Add(); err == nil {
		t.Error("expected error")
	}
}

func TestModify(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	if err := ni.Modify(); err == nil {
		t.Error("expected error")
	}

	icon, err := load()
	if err != nil {
		t.Error(err)
	}
	defer icon.Close()

	ni.Icon = icon
	if err := ni.Add(); err != nil {
		t.Fatal(err)
	}
	if err := ni.Modify(); err != nil {
		t.Error(err)
	}
	// invalid GUID
	windows.MockWindows7()
	ni.GUID = "{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}"
	if err := ni.Modify(); err == nil {
		t.Error("expected error")
	}
}

func TestNotify(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	icon, err := load()
	if err != nil {
		t.Error(err)
	}
	defer icon.Close()

	ni.Icon = icon
	for _, v := range []windows.IconType{
		windows.IconNone,
		windows.IconInfo,
		windows.IconWarn,
		windows.IconError,
		windows.IconUser,
	} {
		n := &windows.Notification{
			Title:    "Title",
			Body:     v.String(),
			IconType: v,
		}
		if err := ni.Notify(n); err != nil {
			t.Error(err)
		}
	}
	n := &windows.Notification{
		Title:    "Title",
		Body:     "IconUser with BalloonIcon",
		IconType: windows.IconUser,
		Icon:     icon,
	}
	if err := ni.Notify(n); err != nil {
		t.Error(err)
	}
}

func TestNotifyError(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	icon, err := load()
	if err != nil {
		t.Error(err)
	}
	defer icon.Close()
	// invalid string
	n := &windows.Notification{
		Title: "\000",
		Body:  "Body",
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
	n = &windows.Notification{
		Title: "Title",
		Body:  "\000",
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
	// NIIF_USER on Windows 2000
	windows.MockWindows2000()
	v := windows.IconUser
	n = &windows.Notification{
		Title:    "Title",
		Body:     v.String(),
		IconType: v,
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
	// NIIF_USER on Windows XP
	windows.MockShellDLLVersion(6, 0, 0)
	v = windows.IconUser
	n = &windows.Notification{
		Title:    "Title",
		Body:     v.String(),
		IconType: v,
		Icon:     icon,
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
	// unknown IconType
	v = windows.IconType(9)
	n = &windows.Notification{
		Title:    "Title",
		Body:     v.String(),
		IconType: v,
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
	// NIIF_NOSOUND on Windows 2000
	windows.MockShellDLLVersion(5, 0, 0)
	n = &windows.Notification{
		Title: "Title",
		Body:  "No Sound",
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
	// invalid GUID
	windows.MockWindows7()
	ni.GUID = "{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}"
	n = &windows.Notification{
		Title: "Title",
		Body:  "Body",
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
}

func load() (*windows.Icon, error) {
	f, err := os.Open(filepath.Join("..", "gopher.png"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return windows.LoadImage(img)
}

func TestIconType(t *testing.T) {
	for i, e := range []string{
		"IconNone",
		"IconInfo",
		"IconWarn",
		"IconError",
		"IconUser",
		"IconType(5)",
	} {
		if g := windows.IconType(i).String(); g != e {
			t.Errorf("IconType.String() = %v, expected %v", g, e)
		}
	}
}

func TestLoadImage(t *testing.T) {
	for _, v := range []reflect.Value{
		reflect.ValueOf(image.NewGray),
		reflect.ValueOf(image.NewNRGBA),
	} {
		rv := v.Call([]reflect.Value{reflect.ValueOf(image.Rect(0, 0, 32, 32))})
		icon, err := windows.LoadImage(rv[0].Interface().(image.Image))
		if err != nil {
			t.Error(err)
		} else {
			icon.Close()
		}
	}

	if _, err := windows.LoadImage(image.NewAlpha(image.Rect(0, 0, 32, 32))); err == nil {
		t.Error("expected error")
	}
}

func TestLoadIcon(t *testing.T) {
	icon, err := windows.LoadIcon(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := icon.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGUID(t *testing.T) {
	e := syscall.GUID{
		Data1: 0x23977b55,
		Data2: 0x10e0,
		Data3: 0x4041,
		Data4: [8]byte{0xb8, 0x62, 0xb1, 0x95, 0x41, 0x96, 0x36, 0x69},
	}
	for _, s := range []string{
		"23977B5510E04041B862B19541963669",
		"23977B55-10E0-4041-B862-B19541963669",
		"{23977B55-10E0-4041-B862-B19541963669}",
	} {
		switch g, err := windows.GUID(s).Parse(); {
		case err != nil:
			t.Error(err)
		case !reflect.DeepEqual(g, e):
			t.Errorf("expected %v, got %v", e, g)
		}
	}
}

func TestGUIDError(t *testing.T) {
	for _, s := range []string{
		"",

		"xxxxxxxx10E04041B862B19541963669",
		"23977B55xxxx4041B862B19541963669",
		"23977B5510E0xxxxB862B19541963669",
		"23977B5510E04041xxxxxxxxxxxxxxxx",

		"{23977B55-10E0-4041-B862-B19541963669",

		"{23977B55-10E0-4041-B862}",
		"{23977B55-10E0-4041-B862B19541963669}",

		"{23977B55-10E0-4041-B862-}",

		"{xxxxxxxx-10E0-4041-B862-B19541963669}",
		"{23977B55-xxxx-4041-B862-B19541963669}",
		"{23977B55-10E0-xxxx-B862-B19541963669}",
		"{23977B55-10E0-4041-xxxx-B19541963669}",
		"{23977B55-10E0-4041-B862-xxxxxxxxxxxx}",
	} {
		if _, err := windows.GUID(s).Parse(); err == nil {
			t.Error("expected error")
		}
	}
}

func TestBalloonEvent(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	for _, nin := range []uintptr{
		sys.NIN_BALLOONSHOW,
		sys.NIN_BALLOONHIDE,
		sys.NIN_BALLOONTIMEOUT,
		sys.NIN_BALLOONUSERCLICK,
	} {
		if err := ni.PostMessage(sys.WM_USER, 0, nin); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Second)

	for _, e := range []windows.BalloonEvent{
		windows.BalloonShown,
		windows.BalloonClosed,
		windows.BalloonClosed,
		windows.BalloonClicked,
	} {
		if g := <-ni.Balloon; !reflect.DeepEqual(g, e) {
			t.Errorf("expected %v, got %v", e, g)
		}
	}

	for i, e := range []string{
		"BalloonShown",
		"BalloonClosed",
		"BalloonClicked",
		"BalloonEvent(3)",
	} {
		if g := windows.BalloonEvent(i).String(); g != e {
			t.Errorf("BalloonEvent.String() = %v, expected %v", g, e)
		}
	}
}

func TestMenu(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	menu := ni.CreateMenu()
	sub := menu.Submenu("Submenu")
	sub.Item("Item 1", 1)
	sub.Sep()
	sub.Item("Item 2", 2)
	menu.Sep()
	menu.Item("Item 3", 3)
	for i := 1; i < 4; i++ {
		if err := ni.PostMessage(sys.WM_USER, 0, sys.WM_RBUTTONUP); err != nil {
			t.Fatal(err)
		}
		if err := ni.PostMessage(sys.WM_COMMAND, uintptr(i), 0); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Second)

	for i := uint16(1); i < 4; i++ {
		if g, e := <-ni.Menu, (windows.MenuEvent{ID: i}); !reflect.DeepEqual(g, e) {
			t.Errorf("expected %#v, got %#v", e, g)
		}
	}
}

func TestMenuError(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	menu := ni.CreateMenu()
	sub := menu.Submenu("Submenu")
	sub.Item("\000", 1)
	if _, err := menu.Sys(); err == nil {
		t.Error("expected error")
	}
}

func TestTaskbarCreated(t *testing.T) {
	ni, err := windows.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ni.Close()

	for i := 0; i < 3; i++ {
		if err := ni.PostMessage(windows.WM_TASKBARCREATED, 0, 0); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Second)
}

func TestVersionError(t *testing.T) {
	for _, s := range []string{
		"XP SP2",
		"Vista",
		"7",
	} {
		err := windows.VersionError(s)
		if !strings.Contains(err.Error(), "Windows "+s) {
			t.Error("unexpected error:", err)
		}
	}
}
