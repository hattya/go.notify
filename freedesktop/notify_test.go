//
// go.notify/freedesktop :: notify_test.go
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

package freedesktop_test

import (
	"image"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/godbus/dbus"
	"github.com/hattya/go.notify/freedesktop"
)

func init() {
	freedesktop.SetSessionBus(func() (*dbus.Conn, error) {
		return dbus.NewConn(new(conn))
	})
}

type conn struct {
}

func (c *conn) Read([]byte) (int, error)  { return 0, io.EOF }
func (c *conn) Write([]byte) (int, error) { return 0, nil }
func (c *conn) Close() error              { return nil }

func TestNewError(t *testing.T) {
	defer func(save func() *dbus.Call) { freedesktop.MockBusMethodCall = save }(freedesktop.MockBusMethodCall)
	freedesktop.MockBusMethodCall = func() *dbus.Call {
		return &dbus.Call{Err: dbus.ErrClosed}
	}
	if _, err := freedesktop.New(); err == nil {
		t.Error("expected error")
	}

	restore := freedesktop.SetSessionBus(func() (*dbus.Conn, error) {
		return nil, dbus.ErrClosed
	})
	defer restore()

	if _, err := freedesktop.New(); err == nil {
		t.Error("expected error")
	}
}

func TestClose(t *testing.T) {
	c, err := freedesktop.New()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCloseNotification(t *testing.T) {
	c, err := freedesktop.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	c.ResetMock()
	c.MockMethodCall(new(dbus.Call))
	if err := c.CloseNotification(1); err != nil {
		t.Fatal(err)
	}
	if g, e := c.NumMethodCalls(), 1; g != e {
		t.Errorf("object calls %v times, expected %v", g, e)
	}

	c.ResetMock()
	c.MockMethodCall(&dbus.Call{Err: dbus.ErrMsgUnknownMethod})
	if err = c.CloseNotification(1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCapabilities(t *testing.T) {
	c, err := freedesktop.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	e := []string{"body", "persistence", "sound"}
	c.ResetMock()
	c.MockMethodCall(&dbus.Call{Body: []interface{}{e}})
	caps, err := c.GetCapabilities()
	if err != nil {
		t.Fatal(err)
	}
	if g, e := c.NumMethodCalls(), 1; g != e {
		t.Errorf("object calls %v times, expected %v", g, e)
	}
	if !reflect.DeepEqual(caps, e) {
		t.Errorf("GetCapabilities = %v, expected %v", caps, e)
	}

	c.ResetMock()
	c.MockMethodCall(&dbus.Call{Err: dbus.ErrMsgUnknownMethod})
	if _, err = c.GetCapabilities(); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetServerInformation(t *testing.T) {
	c, err := freedesktop.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	rv := newServer("1.2")
	c.ResetMock()
	c.MockMethodCall(&dbus.Call{Body: rv})
	si, err := c.GetServerInformation()
	if err != nil {
		t.Fatal(err)
	}
	if g, e := c.NumMethodCalls(), 1; g != e {
		t.Errorf("object calls %v times, expected %v", g, e)
	}
	if g, e := si.Name, rv[0]; g != e {
		t.Errorf("GetServerInformation: name = %v, expected %v", g, e)
	}
	if g, e := si.Vendor, rv[1]; g != e {
		t.Errorf("GetServerInformation: vendor = %v, expected %v", g, e)
	}
	if g, e := si.Version, rv[2]; g != e {
		t.Errorf("GetServerInformation: version = %v, expected %v", g, e)
	}
	if g, e := si.SpecVersion, rv[3]; g != e {
		t.Errorf("GetServerInformation: spec_version = %v, expected %v", g, e)
	}

	c.ResetMock()
	c.MockMethodCall(&dbus.Call{Err: dbus.ErrMsgUnknownMethod})
	if _, err = c.GetServerInformation(); err == nil {
		t.Fatal("expected error")
	}
}

func TestNotify(t *testing.T) {
	c, err := freedesktop.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	for _, tt := range []struct {
		name  string
		value interface{}
	}{
		{"image-data", image.NewGray(image.Rect(0, 0, 48, 48))},
		{"image-path", "path"},
	} {
		for _, ver := range []string{"1.0", "1.1", "1.2"} {
			rv := uint32(1)
			c.ResetMock()
			c.MockMethodCall(&dbus.Call{Body: newServer(ver)})
			c.MockMethodCall(&dbus.Call{Body: []interface{}{rv}})
			n := new(freedesktop.Notification)
			n.Action("default", "Default")
			if err := n.Hint(tt.name, tt.value); err != nil {
				t.Fatal(err)
			}
			id, err := c.Notify(n)
			if err != nil {
				t.Fatal(err)
			}
			if g, e := c.NumMethodCalls(), 2; g != e {
				t.Errorf("object calls %v times, expected %v", g, e)
			}
			if g, e := id, rv; g != e {
				t.Errorf("Notify: id = %v, expected %v", g, e)
			}
		}
		// spec version error
		c.ResetMock()
		c.MockMethodCall(&dbus.Call{Body: newServer("major.minor")})
		n := new(freedesktop.Notification)
		if err := n.Hint(tt.name, tt.value); err != nil {
			t.Fatal(err)
		}
		if _, err = c.Notify(n); err == nil {
			t.Fatal("expected error")
		}
		// server error
		c.ResetMock()
		c.MockMethodCall(&dbus.Call{Err: dbus.ErrMsgUnknownMethod})
		n = new(freedesktop.Notification)
		if err := n.Hint(tt.name, tt.value); err != nil {
			t.Fatal(err)
		}
		if _, err = c.Notify(n); err == nil {
			t.Fatal("expected error")
		}
	}

	// error
	c.ResetMock()
	c.MockMethodCall(&dbus.Call{Err: dbus.ErrMsgUnknownMethod})
	n := new(freedesktop.Notification)
	if _, err = c.Notify(n); err == nil {
		t.Fatal("expected error")
	}
}

func newServer(ver string) []interface{} {
	return []interface{}{"go.notify", "", "0.0", ver}
}

func TestNotificationClosed(t *testing.T) {
	c, err := freedesktop.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	for i := uint32(1); i < 5; i++ {
		c.MockSignal(&dbus.Signal{
			Name: "NotificationClosed",
			Body: []interface{}{i, i},
		})
	}
	for i := uint32(1); i < 5; i++ {
		e := freedesktop.NotificationClosed{
			ID:     i,
			Reason: freedesktop.Reason(i),
		}
		if g := <-c.NotificationClosed; !reflect.DeepEqual(g, e) {
			t.Errorf("<- Client.NotificationClosed = %v, expected %v", g, e)
		}
	}
}

func TestReason(t *testing.T) {
	for i, tt := range []struct {
		s string
		e freedesktop.Reason
	}{
		{"expired", freedesktop.ReasonExpired},
		{"dismissed", freedesktop.ReasonDismissed},
		{"closed", freedesktop.ReasonClosed},
		{"undefined", freedesktop.ReasonUndefined},
	} {
		if g := freedesktop.Reason(i + 1); !strings.Contains(g.String(), tt.s) {
			t.Errorf("Reason.String() = %q, expected %q", g, tt.e)
		}
	}
}

func TestActionInvoked(t *testing.T) {
	c, err := freedesktop.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	for i := uint32(1); i < 5; i++ {
		c.MockSignal(&dbus.Signal{
			Name: "ActionInvoked",
			Body: []interface{}{i, "key"},
		})
	}
	for i := uint32(1); i < 5; i++ {
		e := freedesktop.ActionInvoked{
			ID:  i,
			Key: "key",
		}
		if g := <-c.ActionInvoked; !reflect.DeepEqual(g, e) {
			t.Errorf("<- Client.ActionInvoked = %v, expected %v", g, e)
		}
	}
}

func TestAction(t *testing.T) {
	var n freedesktop.Notification

	e := []string{"default", "Default"}
	n.Action("default", "Default")
	if !reflect.DeepEqual(n.Actions, e) {
		t.Errorf("Notification.Actions = %v, expected %v", n.Actions, e)
	}

	e = []string{"default", "デフォルト"}
	n.Action("default", "デフォルト")
	if !reflect.DeepEqual(n.Actions, e) {
		t.Errorf("Notification.Actions = %v, expected %v", n.Actions, e)
	}
}

func TestHint_ImageData(t *testing.T) {
	for _, v := range []reflect.Value{
		reflect.ValueOf(image.NewGray),
		reflect.ValueOf(image.NewNRGBA),
	} {
		rv := v.Call([]reflect.Value{reflect.ValueOf(image.Rect(0, 0, 48, 48))})
		img := rv[0].Interface().(image.Image)
		data, err := freedesktop.NewImageData(img)
		if err != nil {
			t.Fatal(err)
		}
		e := map[string]interface{}{
			"image-data": data,
		}
		// >= 1.2
		n := new(freedesktop.Notification)
		switch err := n.Hint("image-data", img); {
		case err != nil:
			t.Error(err)
		case !reflect.DeepEqual(n.Hints, e):
			t.Errorf("Notification.Hints = %v, expected %v", n.Hints, e)
		}
		// =  1.1
		n = new(freedesktop.Notification)
		switch err := n.Hint("image_data", data); {
		case err != nil:
			t.Error(err)
		case !reflect.DeepEqual(n.Hints, e):
			t.Errorf("Notification.Hints = %v, expected %v", n.Hints, e)
		}
		// <  1.1
		n = new(freedesktop.Notification)
		switch err := n.Hint("icon_data", *data); {
		case err != nil:
			t.Error(err)
		case !reflect.DeepEqual(n.Hints, e):
			t.Errorf("Notification.Hints = %v, expected %v", n.Hints, e)
		}
	}

	n := new(freedesktop.Notification)
	if err := n.Hint("image-data", image.NewGray16(image.Rect(0, 0, 48, 48))); err == nil {
		t.Error("expected error")
	}
}

func TestHint_ImagePath(t *testing.T) {
	e := map[string]interface{}{
		"image-path": "path",
	}
	var n freedesktop.Notification

	// >= 1.2
	switch err := n.Hint("image-path", "path"); {
	case err != nil:
		t.Error(err)
	case !reflect.DeepEqual(n.Hints, e):
		t.Errorf("Notification.Hints = %v, expected %v", n.Hints, e)
	}
	// =  1.1
	switch err := n.Hint("image_path", "path"); {
	case err != nil:
		t.Error(err)
	case !reflect.DeepEqual(n.Hints, e):
		t.Errorf("Notification.Hints = %v, expected %v", n.Hints, e)
	}
}

func TestHint_X(t *testing.T) {
	testHint_i(t, "x")
}

func TestHint_Y(t *testing.T) {
	testHint_i(t, "y")
}

func testHint_i(t *testing.T, name string) {
	e := map[string]interface{}{
		name: int32(1),
	}

	for _, v := range []interface{}{
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
	} {
		var n freedesktop.Notification
		switch err := n.Hint(name, v); {
		case err != nil:
			t.Errorf("%T: %v", v, err)
		case !reflect.DeepEqual(n.Hints, e):
			t.Errorf("%T: Notification.Hints = %v, expected %v", v, n.Hints, e)
		}
	}

	for _, v := range []interface{}{
		int64(math.MaxInt32 + 1),
		uint64(math.MaxUint32 + 1),
		float32(1),
		float64(1),
		"1",
	} {
		var n freedesktop.Notification
		if err := n.Hint(name, v); err == nil {
			t.Errorf("%T: expected error", v)
		}
	}
}

func TestHint_Urgency(t *testing.T) {
	e := map[string]interface{}{
		"urgency": uint8(1),
	}

	for _, v := range []interface{}{
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
	} {
		var n freedesktop.Notification
		switch err := n.Hint("urgency", v); {
		case err != nil:
			t.Errorf("%T: %v", v, err)
		case !reflect.DeepEqual(n.Hints, e):
			t.Errorf("%T: Notification.Hints = %v, expected %v", v, n.Hints, e)
		}
	}

	for _, v := range []interface{}{
		int64(math.MaxUint8 + 1),
		uint64(math.MaxUint8 + 1),
		float32(1),
		float64(1),
		"1",
	} {
		var n freedesktop.Notification
		if err := n.Hint("urgency", v); err == nil {
			t.Errorf("%T: expected error", v)
		}
	}
}
