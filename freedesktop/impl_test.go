//
// go.notify/freedesktop :: impl_test.go
//
//   Copyright (c) 2017-2019 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package freedesktop_test

import (
	"image"
	"math"
	"testing"

	"github.com/godbus/dbus"
	"github.com/hattya/go.notify"
	"github.com/hattya/go.notify/freedesktop"
)

const name = "go.notify"

func TestNotifierError(t *testing.T) {
	restore := freedesktop.SetSessionBus(func() (*dbus.Conn, error) {
		return nil, dbus.ErrClosed
	})
	defer restore()

	if _, err := freedesktop.NewNotifier(name); err == nil {
		t.Error("expected error")
	}
}

func TestNotifierRegister(t *testing.T) {
	n, err := freedesktop.NewNotifier(name)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	// icon
	for _, icon := range []notify.Icon{
		nil,
		"path",
		image.NewGray(image.Rect(0, 0, 48, 48)),
	} {
		if err := n.Register("event", icon, nil); err != nil {
			t.Error(err)
		}
	}
	// error
	if err := n.Register("event", image.NewAlpha(image.Rect(0, 0, 48, 48)), nil); err == nil {
		t.Error("expected error")
	}
	if err := n.Register("event", 0, nil); err == nil {
		t.Error("expected error")
	}

	for _, opts := range []map[string]interface{}{
		{"freedesktop:actions": map[string]string{"default": "Default"}},
		{"freedesktop:hints": map[string]interface{}{"urgency": 1}},
		{"freedesktop:timeout": 0},
	} {
		if err := n.Register("event", "path", opts); err != nil {
			t.Error(err)
		}
	}
	// error
	for _, opts := range []map[string]interface{}{
		{"freedesktop:actions": nil},
		{"freedesktop:hints": map[string]interface{}{"urgency": math.MaxUint8 + 1}},
		{"freedesktop:hints": nil},
		{"freedesktop:timeout": nil},
	} {
		if err := n.Register("event", "path", opts); err == nil {
			t.Errorf("%v: expected error", opts)
		}
	}
}

func TestNotifierNotify(t *testing.T) {
	n, err := freedesktop.NewNotifier(name)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	c := n.Sys().(*freedesktop.Client)
	c.MockMethodCall(&dbus.Call{Body: []interface{}{uint32(1)}})
	if err := n.Register("event", "path", nil); err != nil {
		t.Fatal(err)
	}
	if err := n.Notify("event", "Title", "Body"); err != nil {
		t.Fatal(err)
	}
	// unknown event
	if err := n.Notify("", "Title", "Body"); err == nil {
		t.Error("expected error")
	}
	// error
	if err := n.Notify("event", "Title", "Body"); err == nil {
		t.Error("expected error")
	}
}
