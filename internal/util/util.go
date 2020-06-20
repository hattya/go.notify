//
// go.notify/internal/util :: util.go
//
//   Copyright (c) 2017-2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package util

import (
	"bufio"
	"bytes"
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
	case *image.NRGBA64:
	case *image.NYCbCrA:
	case *image.Paletted:
	case *image.RGBA:
	case *image.RGBA64:
	case *image.YCbCr:
	default:
		return nil, fmt.Errorf("unsupported image: %T", img)
	}
	var dst draw.Image
	if gray {
		dst = image.NewGray(img.Bounds())
	} else {
		dst = image.NewNRGBA(img.Bounds())
	}
	draw.Draw(dst, dst.Bounds(), img, image.Point{}, draw.Src)
	return dst, nil
}

// ReadBytes reads until the first occurrence of the specified delim in the
// input.
func ReadBytes(r *bufio.Reader, delim []byte) ([]byte, error) {
	var rv []byte
	for {
		b, err := r.ReadBytes(delim[len(delim)-1])
		if err != nil {
			return nil, err
		}
		rv = append(rv, b...)
		if bytes.HasSuffix(rv, delim) {
			break
		}
	}
	return rv, nil
}
