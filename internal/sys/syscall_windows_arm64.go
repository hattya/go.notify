//
// go.notify/internal/sys :: syscall_windows_arm64.go
//
//   Copyright (c) 2017-2019 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package sys

var (
	GetWindowLongPtr = getWindowLongPtr
	SetWindowLongPtr = setWindowLongPtr
)
