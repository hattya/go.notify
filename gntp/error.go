//
// go.notify/gntp :: error.go
//
//   Copyright (c) 2017-2019 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package gntp

import "net/textproto"

// ErrorCode represents an Error-Code value.
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

// Description returns an Error-Description of the Error-Code. It returns the
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
