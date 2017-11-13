//
// go.notify :: notify.go
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

// Package notify provides an interface for notifications.
package notify

import "errors"

var ErrEvent = errors.New("go.notify: unknown event")

// Icon represents an icon. Its value is dependent on each implementation.
type Icon interface{}

// Notifier is an interface for notifications.
type Notifier interface {
	// Close closes the Notifier.
	Close() error

	// Register registers the named event to the Notifier. The keys and values
	// of the opts are dependent on each implementation.
	//
	// Notifier may use the icon for notifications.
	Register(event string, icon Icon, opts map[string]interface{}) error

	// Notify notifies the named event by the specified title and body.
	Notify(event, title, body string) error

	// Sys returns the implementation of the Notifier.
	Sys() interface{}
}
