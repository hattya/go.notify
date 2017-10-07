//
// go.notify/freedesktop :: export_test.go
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

import "github.com/godbus/dbus"

func SetSessionBus(fn func() (*dbus.Conn, error)) func() {
	save := sessionBus
	sessionBus = fn
	return func() { sessionBus = save }
}

func init() {
	testHookNew = func(c *Client) {
		c.obj = &object{
			dest: notifications,
			path: objectPath,
		}
	}
}

func (c *Client) MockMethodCall(call *dbus.Call) {
	obj := c.obj.(*object)
	obj.calls = append(obj.calls, call)
}

func (c *Client) NumMethodCalls() int {
	return c.obj.(*object).n
}

func (c *Client) ResetMock() {
	obj := c.obj.(*object)
	obj.calls = obj.calls[:0]
	obj.n = 0
}

type object struct {
	dest  string
	path  dbus.ObjectPath
	calls []*dbus.Call
	n     int
}

func (o *object) Call(method string, flags dbus.Flags, args ...interface{}) *dbus.Call {
	return o.Go(method, flags, nil, args...)
}

func (o *object) Go(method string, flags dbus.Flags, ch chan *dbus.Call, args ...interface{}) *dbus.Call {
	if len(o.calls) <= o.n {
		return &dbus.Call{Err: dbus.ErrClosed}
	}

	call := o.calls[o.n]
	call.Destination = o.dest
	call.Path = o.path
	call.Method = method
	call.Args = args
	o.n++
	return call
}

func (o *object) GetProperty(p string) (dbus.Variant, error) { return dbus.Variant{}, nil }
func (o *object) Destination() string                        { return o.dest }
func (o *object) Path() dbus.ObjectPath                      { return o.path }
