//
// go.notify/internal/util :: util.go
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

package util

import (
	"fmt"
	"image"
	"image/draw"
)

// Convert converts the specified img to either image.Gray or image.NRGBA.
func Convert(img image.Image) (image.Image, error) {
	var gray bool
	switch img := img.(type) {
	case *image.CMYK:
	case *image.Gray:
		return img, nil
	case *image.Gray16:
		gray = true
	case *image.NRGBA:
		return img, nil
	default:
		return nil, fmt.Errorf("unsupported image: %T", img)
	}
	var dst draw.Image
	if gray {
		dst = image.NewGray(img.Bounds())
	} else {
		dst = image.NewNRGBA(img.Bounds())
	}
	draw.Draw(dst, dst.Bounds(), img, image.ZP, draw.Src)
	return dst, nil
}
