//
// go.notify/gntp :: impl_test.go
//
//   Copyright (c) 2017-2025 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package gntp_test

import (
	"math"
	"testing"

	"github.com/hattya/go.notify/gntp"
)

func TestNotifier(t *testing.T) {
	for _, c := range []*gntp.Client{nil, gntp.New()} {
		n := gntp.NewNotifier(c)
		if v := n.Sys().(*gntp.Client); v == nil {
			t.Errorf("expected *Client, got %v", v)
		}
	}
}

func TestNotifierRegister(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name
	n := gntp.NewNotifier(c)
	defer n.Close()

	for _, opts := range []map[string]any{
		{"gntp:display-name": "Display Name"},
		{"gntp:enabled": true},
		{"gntp:sticky": true},
		{"gntp:priority": int(1)},
		{"gntp:priority": int8(1)},
		{"gntp:priority": int16(1)},
		{"gntp:priority": int32(1)},
		{"gntp:priority": int64(1)},
		{"gntp:priority": uint(1)},
		{"gntp:priority": uint8(1)},
		{"gntp:priority": uint16(1)},
		{"gntp:priority": uint32(1)},
		{"gntp:priority": uint64(1)},
	} {
		s.MockOK("REGISTER", gntp.NONE)
		if err := n.Register("event", "path", opts); err != nil {
			t.Error(err)
		}
	}
	// error
	for _, opts := range []map[string]any{
		{"gntp:display-name": nil},
		{"gntp:enabled": nil},
		{"gntp:sticky": nil},
		{"gntp:priority": nil},
		{"gntp:priority": int64(math.MaxInt32 + 1)},
		{"gntp:priority": uint64(math.MaxInt32 + 1)},
		{"gntp:priority": float32(1)},
		{"gntp:priority": float64(1)},
		{"gntp:priority": "1"},
	} {
		if err := n.Register("event", "path", opts); err == nil {
			t.Errorf("%v: expected error", opts)
		}
	}
}

func TestNotifierNotify(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name
	n := gntp.NewNotifier(c)
	defer n.Close()

	s.MockOK("REGISTER", gntp.NONE)
	if err := n.Register("event", "path", nil); err != nil {
		t.Error(err)
	}
	s.MockOK("NOTIFY", gntp.NONE)
	if err := n.Notify("event", "Title", "Body"); err != nil {
		t.Fatal(err)
	}
	// unknown event
	if err := n.Notify("", "Title", "Body"); err == nil {
		t.Error("expected error")
	}

	c = n.Sys().(*gntp.Client)
	c.Wait()
}
