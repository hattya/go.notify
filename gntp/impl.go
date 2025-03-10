//
// go.notify/gntp :: impl.go
//
//   Copyright (c) 2017-2025 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package gntp

import (
	"fmt"
	"math"

	"github.com/hattya/go.notify"
)

type notifier struct {
	c  *Client
	ev map[string]*Notification
}

// NewNotifier returns a new Notifier.
//
// Register supports following icon types:
//   - string
//   - []byte
//   - image.Image
//   - io.Reader
//
// Register accepts following keys and values types:
//   - gntp:display-name string
//   - gntp:enabled      bool
//   - gntp:sticky       bool
//   - gntp:priority     int
func NewNotifier(c *Client) notify.Notifier {
	if c == nil {
		c = New()
	}
	return &notifier{
		c:  c,
		ev: make(map[string]*Notification),
	}
}

func (p *notifier) Close() error {
	p.c.Reset()
	return nil
}

func (p *notifier) Notify(event, title, body string) error {
	n := new(Notification)
	if ev, ok := p.ev[event]; ok {
		*n = *ev
	} else {
		return notify.ErrEvent
	}
	n.Title = title
	n.Text = body
	_, err := p.c.Notify(n)
	return err
}

func (p *notifier) Register(event string, icon notify.Icon, opts map[string]any) error {
	n := &Notification{
		Name:    event,
		Enabled: true,
		Icon:    icon,
	}
	k := "gntp:display-name"
	if v, ok := opts[k]; ok {
		if s, ok := v.(string); ok {
			n.DisplayName = s
		} else {
			return fmt.Errorf("%q expects string: %T", k, v)
		}
	}
	k = "gntp:enabled"
	if v, ok := opts[k]; ok {
		if b, ok := v.(bool); ok {
			n.Enabled = b
		} else {
			return fmt.Errorf("%q expects bool: %T", k, v)
		}
	}
	k = "gntp:sticky"
	if v, ok := opts[k]; ok {
		if b, ok := v.(bool); ok {
			n.Sticky = b
		} else {
			return fmt.Errorf("%q expects bool: %T", k, v)
		}
	}
	k = "gntp:priority"
	if v, ok := opts[k]; ok {
		i2i := func(i int64) (int, bool) {
			if math.MinInt32 <= i && i <= math.MaxInt32 {
				return int(i), true
			}
			return 0, false
		}
		u2i := func(u uint64) (int, bool) {
			if u <= math.MaxInt32 {
				return int(u), true
			}
			return 0, false
		}
		var i int
		var ok bool
		switch v := v.(type) {
		case int:
			i, ok = v, true
		case int8:
			i, ok = i2i(int64(v))
		case int16:
			i, ok = i2i(int64(v))
		case int32:
			i, ok = i2i(int64(v))
		case int64:
			i, ok = i2i(v)
		case uint:
			i, ok = u2i(uint64(v))
		case uint8:
			i, ok = u2i(uint64(v))
		case uint16:
			i, ok = u2i(uint64(v))
		case uint32:
			i, ok = u2i(uint64(v))
		case uint64:
			i, ok = u2i(v)
		default:
			return fmt.Errorf("%q expects int: %T", k, v)
		}
		if !ok {
			return fmt.Errorf("%q overflows int range: %v", k, v)
		}
		n.Priority = i
	}
	p.ev[event] = n

	list := make([]*Notification, len(p.ev))
	i := 0
	for _, n := range p.ev {
		list[i] = n
		i++
	}
	_, err := p.c.Register(list)
	return err
}

func (p *notifier) Sys() any {
	return p.c
}
