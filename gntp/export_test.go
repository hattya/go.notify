//
// go.notify/gntp :: export_test.go
//
//   Copyright (c) 2017-2019 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package gntp

import "crypto/cipher"

var RFC3339 = rfc3339

func (c *Client) Send(mt string) (resp *Response, err error) {
	return c.send(mt, c.buffer())
}

func (c *Client) Wait() {
	c.wg.Wait()
}

func (i *Info) Cipher() cipher.Block {
	return i.cipher
}
