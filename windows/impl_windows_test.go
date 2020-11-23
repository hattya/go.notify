//
// go.notify/windows :: impl_windows_test.go
//
//   Copyright (c) 2017-2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package windows_test

import (
	"image"
	"math"
	"os"
	"testing"
	"time"

	"github.com/hattya/go.notify"
	"github.com/hattya/go.notify/windows"
)

func TestNotifierError(t *testing.T) {
	if _, err := windows.NewNotifier("\000", nil); err == nil {
		t.Error("expected error")
	}
}

func TestNotifierRegister(t *testing.T) {
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
		"windows:sound": false,
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

func TestNotifierNotify(t *testing.T) {
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
			switch {
			case os.Getenv("GITHUB_ACTIONS") != "":
				// Windows Push Notifications User Service is disabled
			case i < 1:
				t.Fatal("timeout")
			}
			i = 9
		}
	}
}
