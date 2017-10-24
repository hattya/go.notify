//
// go.notify/windows :: impl_windows_test.go
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
	"math"
	"testing"
	"time"

	"github.com/hattya/go.notify"
	"github.com/hattya/go.notify/windows"
)

func TestNotiferError(t *testing.T) {
	if _, err := windows.NewNotifier("\000", nil); err == nil {
		t.Error("expected error")
	}
}

func TestNotiferRegister(t *testing.T) {
	n, err := windows.NewNotifier(name, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	icon, err := windows.LoadImage(image.NewGray(image.Rect(0, 0, 32, 32)))
	if err != nil {
		t.Fatal(err)
	}

	// icon
	for _, icon := range []notify.Icon{
		nil,
		image.NewGray(image.Rect(0, 0, 32, 32)),
		int(1),
		int8(1),
		int16(1),
		int32(1),
		int64(1),
		uint(1),
		uint8(1),
		uint16(1),
		uint32(1),
		uint64(1),
		windows.IconInfo,
		icon,
	} {
		if err := n.Register("event", icon, nil); err != nil {
			t.Error(err)
		}
	}
	// error
	for _, icon := range []notify.Icon{
		image.NewAlpha(image.Rect(0, 0, 32, 32)),
		int64(math.MaxUint16 + 1),
		uint64(math.MaxUint16 + 1),
		float32(1),
		float64(1),
		"1",
	} {
		if err := n.Register("event", icon, nil); err == nil {
			t.Error("expected error")
		}
	}

	// windows:fallback-icon
	windows.MockShellDLLVersion(6, 0, 0)
	opts := map[string]interface{}{
		"windows:fallback-icon": windows.IconInfo,
	}
	if err := n.Register("event", icon, opts); err != nil {
		t.Error(err)
	}
	// error
	windows.MockShellDLLVersion(6, 0, 0)
	opts = map[string]interface{}{
		"windows:fallback-icon": 1,
	}
	if err := n.Register("event", icon, opts); err == nil {
		t.Error("expected error")
	}

	// windows:sound
	opts = map[string]interface{}{
		"windows:sound": true,
	}
	if err := n.Register("event", nil, opts); err != nil {
		t.Error(err)
	}
	windows.MockShellDLLVersion(5, 0, 0)
	opts = map[string]interface{}{
		"windows:sound": false,
	}
	if err := n.Register("event", nil, opts); err != nil {
		t.Error(err)
	}
	// error
	opts = map[string]interface{}{
		"windows:sound": 1,
	}
	if err := n.Register("event", nil, opts); err == nil {
		t.Error("expected error")
	}
}

func TestNotiferNotify(t *testing.T) {
	n, err := windows.NewNotifier(name, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	opts := map[string]interface{}{
		"windows:sound": false,
	}
	if err := n.Register("event", windows.IconInfo, opts); err != nil {
		t.Fatal(err)
	}
	if err := n.Notify("event", "Title", "Body"); err != nil {
		t.Fatal(err)
	}
	// unknown event
	if err := n.Notify("", "Title", "Body"); err == nil {
		t.Error("expected error")
	}

	ni := n.Sys().(*windows.NotifyIcon)
	for i := 0; i < 2; {
		select {
		case <-ni.Balloon:
			i++
		case <-time.After(1 * time.Second):
			if i < 1 {
				t.Fatal("timeout")
			}
			i = 99
		}
	}
}
