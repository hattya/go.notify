//
// go.notify/internal/util :: util_test.go
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

package util_test

import (
	"image"
	"image/color/palette"
	"reflect"
	"testing"

	"github.com/hattya/go.notify/internal/util"
)

func TestConvert(t *testing.T) {
	for _, v := range []reflect.Value{
		reflect.ValueOf(image.NewCMYK),
		reflect.ValueOf(image.NewGray),
		reflect.ValueOf(image.NewGray16),
		reflect.ValueOf(image.NewNRGBA),
		reflect.ValueOf(image.NewNRGBA64),
		reflect.ValueOf(func(r image.Rectangle) image.Image { return image.NewNYCbCrA(r, image.YCbCrSubsampleRatio420) }),
		reflect.ValueOf(func(r image.Rectangle) image.Image { return image.NewPaletted(r, palette.Plan9) }),
		reflect.ValueOf(image.NewRGBA),
	} {
		rv := v.Call([]reflect.Value{reflect.ValueOf(image.Rect(0, 0, 32, 32))})
		img := rv[0].Interface().(image.Image)
		if _, err := util.Convert(img); err != nil {
			t.Error(err)
		}
	}

	if _, err := util.Convert(image.NewAlpha(image.Rect(0, 0, 32, 32))); err == nil {
		t.Error("expected error")
	}
}
