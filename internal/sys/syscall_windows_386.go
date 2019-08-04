//
// go.notify/internal/sys :: syscall_windows_386.go
//
//   Copyright (c) 2017-2019 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package sys

var (
	GetWindowLongPtr = getWindowLong
	SetWindowLongPtr = setWindowLong
)
