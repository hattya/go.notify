//
// go.notify/freedesktop :: export_test.go
//
//   Copyright (c) 2017-2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package freedesktop

import (
	"context"

	"github.com/godbus/dbus/v5"
)

func SetSessionBus(fn func() (*dbus.Conn, error)) func() {
	save := sessionBus
	sessionBus = fn
	return func() { sessionBus = save }
}

var MockBusMethodCall = func() *dbus.Call { return new(dbus.Call) }

func init() {
	testHookNew = func(c *Client) {
		c.busObj = &object{
			dest:  c.busObj.Destination(),
			path:  c.busObj.Path(),
			calls: []*dbus.Call{MockBusMethodCall(), MockBusMethodCall()},
		}
		c.obj = &object{
			dest: iface,
			path: path,
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

func (c *Client) MockSignal(sig *dbus.Signal) {
	sig.Path = path
	sig.Name = iface + "." + sig.Name
	c.c <- sig
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
	return o.GoWithContext(context.Background(), method, flags, nil, args...)
}

func (o *object) CallWithContext(ctx context.Context, method string, flags dbus.Flags, args ...interface{}) *dbus.Call {
	return o.GoWithContext(ctx, method, flags, nil, args...)
}

func (o *object) Go(method string, flags dbus.Flags, ch chan *dbus.Call, args ...interface{}) *dbus.Call {
	return o.GoWithContext(context.Background(), method, flags, ch, args...)
}

func (o *object) GoWithContext(ctx context.Context, method string, flags dbus.Flags, ch chan *dbus.Call, args ...interface{}) *dbus.Call {
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

func (o *object) AddMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	return nil
}

func (o *object) RemoveMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	return nil
}

func (o *object) GetProperty(p string) (dbus.Variant, error) { return dbus.Variant{}, nil }
func (o *object) SetProperty(p string, v interface{}) error  { return nil }
func (o *object) Destination() string                        { return o.dest }
func (o *object) Path() dbus.ObjectPath                      { return o.path }
