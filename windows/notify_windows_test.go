//
// go.notify/windows :: notify_windows_test.go
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

package windows_test

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hattya/go.notify/windows"
	syscall "golang.org/x/sys/windows"
)

const (
	name = "go.notify"
	guid = "{23977B55-10E0-4041-B862-B19541963669}"
)

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
	ni.GUID = guid
	if err := ni.Add(); err != nil {
		t.Error(err)
	}
	// already added
	if err := ni.Add(); err == nil {
		t.Error("expected error")
	}
	// invalid GUID
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
	ni.GUID = guid
	if err := ni.Add(); err != nil {
		t.Fatal(err)
	}
	if err := ni.Modify(); err != nil {
		t.Error(err)
	}
	// invalid GUID
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
	ni.GUID = guid
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
	// unknown IconType
	v := windows.IconType(9)
	n = &windows.Notification{
		Title:    "Title",
		Body:     v.String(),
		IconType: v,
	}
	if err := ni.Notify(n); err == nil {
		t.Error("expected error")
	}
	// invalid GUID
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
