//
// go.notify/freedesktop :: impl.go
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

package freedesktop

import (
	"fmt"
	"image"

	"github.com/hattya/go.notify"
)

type notifier struct {
	c    *Client
	name string
	ev   map[string]*Notification
}

// NewNotifier returns a new Notifier.
//
// Register supports following icon types:
//   - string
//   - image.Image
//
// Register accepts following keys and value types:
//   - freedesktop:actions map[string]string
//   - freedesktop:hints   map[string]interface{}
//   - freedesktop:timeout int32
func NewNotifier(name string) (notify.Notifier, error) {
	c, err := New()
	if err != nil {
		return nil, err
	}
	return &notifier{
		c:    c,
		name: name,
		ev:   make(map[string]*Notification),
	}, nil
}

func (p *notifier) Close() error {
	return p.c.Close()
}

func (p *notifier) Register(event string, icon notify.Icon, opts map[string]interface{}) error {
	n := new(Notification)
	switch icon := icon.(type) {
	case nil:
	case string:
		n.Icon = icon
	case image.Image:
		if err := n.Hint("image-data", icon); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported icon: %T", icon)
	}
	k := "freedesktop:actions"
	if v, ok := opts[k]; ok {
		if m, ok := v.(map[string]string); ok {
			for k, v := range m {
				n.Action(k, v)
			}
		} else {
			return fmt.Errorf("%q expects map[string]string: %T", k, v)
		}
	}
	k = "freedesktop:hints"
	if v, ok := opts[k]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			for k, v := range m {
				if err := n.Hint(k, v); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("%q expects map[string]interface{}: %T", k, v)
		}
	}
	k = "freedesktop:timeout"
	if v, ok := opts[k]; ok {
		if i, err := v2i(k, v); err == nil {
			n.Timeout = i
		} else {
			return fmt.Errorf("%q expects int32: %T", k, v)
		}
	}
	p.ev[event] = n
	return nil
}

func (p *notifier) Notify(event, title, body string) error {
	n := new(Notification)
	if ev, ok := p.ev[event]; ok {
		*n = *ev
	} else {
		return notify.ErrEvent
	}
	n.Name = p.name
	n.Summary = title
	n.Body = body
	_, err := p.c.Notify(n)
	return err
}

func (p *notifier) Sys() interface{} {
	return p.c
}
