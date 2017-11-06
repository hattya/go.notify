//
// go.notify/gntp :: error.go
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

package gntp

import "net/textproto"

// ErrorCode represents a Error-Code value.
type ErrorCode int

// List of Error-Code values.
const (
	TimedOut               ErrorCode = 200
	NetworkFailure         ErrorCode = 201
	InvalidRequest         ErrorCode = 300
	UnknownProtocol        ErrorCode = 301
	UnknownProtocolVersion ErrorCode = 302
	RequiredHeaderMissing  ErrorCode = 303
	NotAuthorized          ErrorCode = 400
	UnknownApplication     ErrorCode = 401
	UnknownNotification    ErrorCode = 402
	AlreadyProcessed       ErrorCode = 403
	NotificationDisabled   ErrorCode = 404
	InternalServerError    ErrorCode = 500
)

// Description returns a Error-Description of the Error-Code. It returns the
// empty string if the code is unknown.
func (code ErrorCode) Description() string {
	return errorDescription[code]
}

var errorDescription = map[ErrorCode]string{
	TimedOut:               "Timed Out",
	NetworkFailure:         "Network Failure",
	InvalidRequest:         "Invalid Request",
	UnknownProtocol:        "Unknown Protocol",
	UnknownProtocolVersion: "Unknown Protocol Version",
	RequiredHeaderMissing:  "Required Header Missing",
	NotAuthorized:          "Not Authorized",
	UnknownApplication:     "Unknown Application",
	UnknownNotification:    "Unknown Notification",
	AlreadyProcessed:       "Already Processed",
	NotificationDisabled:   "Notification Disabled",
	InternalServerError:    "Internal Server Error",
}

// Error represents a GNTP error.
type Error struct {
	Code        ErrorCode
	Description string
	Header      textproto.MIMEHeader
}

func (e Error) Error() string {
	if e.Description != "" {
		return e.Description
	}
	return e.Code.Description()
}
