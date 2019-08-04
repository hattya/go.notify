//
// go.notify/internal/util :: util_test.go
//
//   Copyright (c) 2017-2019 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package util_test

import (
	"bufio"
	"bytes"
	"image"
	"image/color/palette"
	"io"
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
		reflect.ValueOf(image.NewRGBA64),
		reflect.ValueOf(func(r image.Rectangle) image.Image { return image.NewYCbCr(r, image.YCbCrSubsampleRatio420) }),
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

func TestReadBytes(t *testing.T) {
	if _, err := util.ReadBytes(bufio.NewReader(new(bytes.Buffer)), []byte("\r\n")); err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}

	e := []byte("data\r\n")
	switch g, err := util.ReadBytes(bufio.NewReader(bytes.NewReader(e)), []byte("\r\n")); {
	case err != nil:
		t.Error(err)
	case !reflect.DeepEqual(g, e):
		t.Errorf("expected %v, got %v", e, g)
	}
}
