//
// go.notify/windows :: impl_windows.go
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
	"fmt"
	"image"
	"math"

	"github.com/hattya/go.notify"
)

type notifier struct {
	ni *NotifyIcon
	ev map[string]*Notification
}

// NewNotifier returns a new Notifier.
//
// Register supports following icon types:
//   - image.Image
//   - uint16
//   - IconType
//   - *Icon
//
// Register accepts following keys and value types:
//   - windows:fallback-icon IconType
//     This is used on Windows XP or earlier if the specified icon is *Icon.
//   - windows:sound         bool
//     This is ignored on Windows 2000 or earlier.
func NewNotifier(name string, icon *Icon) (notify.Notifier, error) {
	ni, err := New(name)
	if err != nil {
		return nil, err
	}
	ni.Icon = icon
	return &notifier{
		ni: ni,
		ev: make(map[string]*Notification),
	}, nil
}

func (p *notifier) Close() error {
	return p.ni.Close()
}

func (p *notifier) Register(event string, icon notify.Icon, opts map[string]interface{}) error {
	loadIconI := func(i int64) (notify.Icon, error) {
		if 0 <= i && i <= math.MaxUint16 {
			return LoadIcon(uint16(i))
		}
		return i, nil
	}
	loadIconU := func(u uint64) (notify.Icon, error) {
		if u <= math.MaxUint16 {
			return LoadIcon(uint16(u))
		}
		return u, nil
	}
	var err error
	switch v := icon.(type) {
	case image.Image:
		icon, err = LoadImage(v)
	case int:
		icon, err = loadIconI(int64(v))
	case int8:
		icon, err = loadIconI(int64(v))
	case int16:
		icon, err = loadIconI(int64(v))
	case int32:
		icon, err = loadIconI(int64(v))
	case int64:
		icon, err = loadIconI(v)
	case uint:
		icon, err = loadIconU(uint64(v))
	case uint8:
		icon, err = loadIconU(uint64(v))
	case uint16:
		icon, err = loadIconU(uint64(v))
	case uint32:
		icon, err = loadIconU(uint64(v))
	case uint64:
		icon, err = loadIconU(v)
	}
	if err != nil {
		return err
	}

	n := new(Notification)
	switch icon := icon.(type) {
	case nil:
	case IconType:
		n.IconType = icon
	case *Icon:
		if isShellDLLVersionOrGreater(6, 0, 6) {
			n.Icon = icon
			n.IconType = IconUser
		} else {
			k := "windows:fallback-icon"
			if v, ok := opts[k]; ok {
				if icon, ok := v.(IconType); ok {
					n.IconType = icon
				} else {
					return fmt.Errorf("%q expects IconType: %T", k, v)
				}
			}
		}
	default:
		return fmt.Errorf("unsupported icon: %T", icon)
	}
	k := "windows:sound"
	if v, ok := opts[k]; ok {
		if b, ok := v.(bool); ok {
			if isShellDLLVersionOrGreater(6, 0, 0) {
				n.Sound = b
			} else {
				n.Sound = true
			}
		} else {
			return fmt.Errorf("%q expects bool: %T", k, v)
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
	n.Title = title
	n.Body = body
	return p.ni.Notify(n)
}

func (p *notifier) Sys() interface{} {
	return p.ni
}
