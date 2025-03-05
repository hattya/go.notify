//
// go.notify/gntp :: gntp_test.go
//
//   Copyright (c) 2017-2025 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package gntp_test

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"image"
	"io"
	"net"
	"net/textproto"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/hattya/go.notify/gntp"
)

const (
	name     = "go.notify"
	password = "password"
)

func TestRegister(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name

	for _, tt := range []struct {
		auth       bool
		hash       gntp.HashAlgorithm
		encryption gntp.EncryptionAlgorithm
	}{
		// plain text
		{},
		// auth
		{true, gntp.MD5, gntp.NONE},
		{true, gntp.SHA1, gntp.NONE},
		{true, gntp.SHA256, gntp.NONE},
		{true, gntp.SHA512, gntp.NONE},
		// encrypt
		{true, gntp.MD5, gntp.DES},
		{true, gntp.SHA1, gntp.DES},
		{true, gntp.SHA256, gntp.DES},
		{true, gntp.SHA512, gntp.DES},
		{true, gntp.SHA256, gntp.TDES},
		{true, gntp.SHA512, gntp.TDES},
		{true, gntp.SHA256, gntp.AES},
		{true, gntp.SHA512, gntp.AES},
	} {
		if tt.auth {
			s.SetPassword(password)
			c.Password = password
		} else {
			s.SetPassword("")
			c.Password = ""
		}
		c.HashAlgorithm = tt.hash
		c.EncryptionAlgorithm = tt.encryption
		s.MockOK("REGISTER", gntp.NONE)
		_, err := c.Register([]*gntp.Notification{
			{
				Name:        "Name",
				DisplayName: "Display Name",
				Enabled:     true,
			},
		})
		if err != nil {
			t.Error(err)
		}
	}
	// icon
	url := "https://example.com/gopher.png"
	b, err := os.ReadFile(filepath.Join("..", "gopher.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		encrypt bool
		icon    []gntp.Icon
	}{
		// plain text
		{false, []gntp.Icon{url, url}},
		{false, []gntp.Icon{bytes.NewReader(b), bytes.NewReader(b)}},
		{false, []gntp.Icon{b, b}},
		{false, []gntp.Icon{img, img}},
		// encrypt
		{true, []gntp.Icon{url, url}},
		{true, []gntp.Icon{bytes.NewReader(b), bytes.NewReader(b)}},
		{true, []gntp.Icon{b, b}},
		{false, []gntp.Icon{img, img}},
	} {
		if tt.encrypt {
			s.SetPassword(password)
			c.Password = password
			c.HashAlgorithm = gntp.SHA256
			c.EncryptionAlgorithm = gntp.AES
		} else {
			s.SetPassword("")
			c.Password = ""
			c.HashAlgorithm = gntp.MD5
			c.EncryptionAlgorithm = gntp.NONE
		}
		s.MockOK("REGISTER", gntp.NONE)
		c.Icon = tt.icon[0]
		_, err := c.Register([]*gntp.Notification{
			{
				Name:        "Name",
				DisplayName: "Display Name",
				Enabled:     true,
				Icon:        tt.icon[1],
			},
		})
		if err != nil {
			t.Error(err)
		}
	}
	// custom headers
	c.Icon = nil
	for _, tt := range []struct {
		encrypt bool
		value   interface{}
	}{
		// plain text
		{false, "string"},
		{false, []byte("[]byte")},
		{false, bytes.NewReader([]byte("io.Reader"))},
		// encrypt
		{true, "string"},
		{true, []byte("[]byte")},
		{true, bytes.NewReader([]byte("io.Reader"))},
	} {
		if tt.encrypt {
			s.SetPassword(password)
			c.Password = password
			c.HashAlgorithm = gntp.SHA256
			c.EncryptionAlgorithm = gntp.AES
		} else {
			s.SetPassword("")
			c.Password = ""
			c.HashAlgorithm = gntp.MD5
			c.EncryptionAlgorithm = gntp.NONE
		}
		s.MockOK("REGISTER", gntp.NONE)
		c.Header["X-Header"] = tt.value
		_, err := c.Register([]*gntp.Notification{
			{
				Name:        "Name",
				DisplayName: "Display Name",
				Enabled:     true,
			},
		})
		if err != nil {
			t.Error(err)
		}
	}
}

func TestRegisterError(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name

	// unsupported icon
	c.Icon = 0
	if _, err := c.Register(nil); err == nil {
		t.Error("expected error")
	}
	c.Icon = nil
	if _, err := c.Register([]*gntp.Notification{{Icon: 0}}); err == nil {
		t.Error("expected error")
	}
	// image error
	for _, img := range []image.Image{
		image.NewAlpha(image.Rect(0, 0, 32, 32)),
		image.NewGray(image.Rect(0, 0, 0, 0)),
	} {
		c.Icon = img
		if _, err := c.Register(nil); err == nil {
			t.Error("expected error")
		}
	}
	// read error
	c.Icon = new(reader)
	if _, err := c.Register(nil); err != io.ErrUnexpectedEOF {
		t.Errorf("expected io.ErrUnexpectedEOF, got %v", err)
	}
	c.Icon = nil
	c.Header["X-Header"] = new(reader)
	if _, err := c.Register(nil); err != io.ErrUnexpectedEOF {
		t.Errorf("expected io.ErrUnexpectedEOF, got %v", err)
	}
	delete(c.Header, "X-Header")
	if _, err := c.Register([]*gntp.Notification{{Icon: new(reader)}}); err != io.ErrUnexpectedEOF {
		t.Errorf("expected io.ErrUnexpectedEOF, got %v", err)
	}
	// unknown hash algorithm
	c.Icon = []byte("[]byte")
	c.HashAlgorithm = -1
	if _, err := c.Register(nil); err != gntp.ErrHash {
		t.Errorf("expected ErrHash, got %v", err)
	}
}

func TestNotify(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name

	s.MockOK("REGISTER", gntp.NONE)
	_, err := c.Register([]*gntp.Notification{
		{
			Name:    "Name",
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		auth       bool
		hash       gntp.HashAlgorithm
		encryption gntp.EncryptionAlgorithm
	}{
		// plain text
		{},
		// auth
		{true, gntp.MD5, gntp.NONE},
		{true, gntp.SHA1, gntp.NONE},
		{true, gntp.SHA256, gntp.NONE},
		{true, gntp.SHA512, gntp.NONE},
		// encrypt
		{true, gntp.MD5, gntp.DES},
		{true, gntp.SHA1, gntp.DES},
		{true, gntp.SHA256, gntp.DES},
		{true, gntp.SHA512, gntp.DES},
		{true, gntp.SHA256, gntp.TDES},
		{true, gntp.SHA512, gntp.TDES},
		{true, gntp.SHA256, gntp.AES},
		{true, gntp.SHA512, gntp.AES},
	} {
		if tt.auth {
			s.SetPassword(password)
			c.Password = password
		} else {
			s.SetPassword("")
			c.Password = ""
		}
		c.HashAlgorithm = tt.hash
		c.EncryptionAlgorithm = tt.encryption
		s.MockOK("NOTIFY", gntp.NONE)
		_, err := c.Notify(&gntp.Notification{
			Name:           "Name",
			ID:             "ID",
			Sticky:         true,
			Priority:       2,
			CoalescingID:   "CoalescingID",
			CallbackTarget: "CallbackTarget",
		})
		if err != nil {
			t.Error(err)
		}
	}
	// icon
	url := "https://example.com/gopher.png"
	b, err := os.ReadFile(filepath.Join("..", "gopher.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		encrypt bool
		icon    gntp.Icon
	}{
		// plain text
		{false, url},
		{false, bytes.NewReader(b)},
		{false, b},
		{false, img},
		// encrypt
		{true, url},
		{true, bytes.NewReader(b)},
		{true, b},
		{true, img},
	} {
		if tt.encrypt {
			s.SetPassword(password)
			c.Password = password
			c.HashAlgorithm = gntp.SHA256
			c.EncryptionAlgorithm = gntp.AES
		} else {
			s.SetPassword("")
			c.Password = ""
			c.HashAlgorithm = gntp.MD5
			c.EncryptionAlgorithm = gntp.NONE
		}
		s.MockOK("NOTIFY", gntp.NONE)
		c.Icon = tt.icon
		_, err := c.Notify(&gntp.Notification{
			Name:           "Name",
			ID:             "ID",
			Sticky:         true,
			Priority:       2,
			Icon:           tt.icon,
			CoalescingID:   "CoalescingID",
			CallbackTarget: "CallbackTarget",
		})
		if err != nil {
			t.Error(err)
		}
	}
	// custom headers
	c.Icon = nil
	for _, tt := range []struct {
		encrypt bool
		value   interface{}
	}{
		// plain text
		{false, "string"},
		{false, []byte("[]byte")},
		{false, bytes.NewReader([]byte("io.Reader"))},
		// encrypt
		{true, "string"},
		{true, []byte("[]byte")},
		{true, bytes.NewReader([]byte("io.Reader"))},
	} {
		if tt.encrypt {
			s.SetPassword(password)
			c.Password = password
			c.HashAlgorithm = gntp.SHA256
			c.EncryptionAlgorithm = gntp.AES
		} else {
			s.SetPassword("")
			c.Password = ""
			c.HashAlgorithm = gntp.MD5
			c.EncryptionAlgorithm = gntp.NONE
		}
		s.MockOK("NOTIFY", gntp.NONE)
		c.Header["X-Header"] = tt.value
		_, err := c.Notify(&gntp.Notification{
			Name:           "Name",
			ID:             "ID",
			Sticky:         true,
			Priority:       2,
			CoalescingID:   "CoalescingID",
			CallbackTarget: "CallbackTarget",
		})
		if err != nil {
			t.Error(err)
		}
	}
}

func TestNotifyError(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name

	s.MockOK("REGISTER", gntp.NONE)
	_, err := c.Register([]*gntp.Notification{
		{
			Name:    "Name",
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// unsupported icon
	_, err = c.Notify(&gntp.Notification{
		Name:  "Name",
		Title: "Title",
		Text:  "Text",
		Icon:  0,
	})
	if err == nil {
		t.Error("expected error")
	}
	// image error
	for _, img := range []image.Image{
		image.NewAlpha(image.Rect(0, 0, 32, 32)),
		image.NewGray(image.Rect(0, 0, 0, 0)),
	} {
		_, err = c.Notify(&gntp.Notification{
			Name:  "Name",
			Title: "Title",
			Text:  "Text",
			Icon:  img,
		})
		if err == nil {
			t.Error("expected error")
		}
	}
	// read error
	_, err = c.Notify(&gntp.Notification{
		Name:  "Name",
		Title: "Title",
		Text:  "Text",
		Icon:  new(reader),
	})
	if err != io.ErrUnexpectedEOF {
		t.Errorf("expected io.ErrUnexpectedEOF, got %v", err)
	}
	c.Header["X-Header"] = new(reader)
	_, err = c.Notify(&gntp.Notification{
		Name:  "Name",
		Title: "Title",
		Text:  "Text",
	})
	if err != io.ErrUnexpectedEOF {
		t.Errorf("expected io.ErrUnexpectedEOF, got %v", err)
	}
	delete(c.Header, "X-Header")
	// unknown hash algorithm
	c.HashAlgorithm = -1
	_, err = c.Notify(&gntp.Notification{
		Name:  "Name",
		Title: "Title",
		Text:  "Text",
		Icon:  []byte("[]byte"),
	})
	if err != gntp.ErrHash {
		t.Errorf("expected ErrHash, got %v", err)
	}
}

type reader struct {
}

func (r *reader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestCallback(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name

	s.MockOK("REGISTER", gntp.NONE)
	_, err := c.Register([]*gntp.Notification{
		{
			Name:    "Name",
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, ea := range []gntp.EncryptionAlgorithm{
		gntp.NONE,
		gntp.DES,
		gntp.TDES,
		gntp.AES,
	} {
		for i := range 10 {
			if ea != gntp.NONE {
				s.SetPassword(password)
				c.Password = password
			} else {
				s.SetPassword("")
				c.Password = ""
			}
			res := gntp.Result((i % 3) + 1)
			s.MockCallback(res, ea)
			_, err := c.Notify(&gntp.Notification{
				Name:                "Name",
				CallbackContext:     strconv.Itoa(i),
				CallbackContextType: "int",
			})
			if err != nil {
				t.Error(err)
			} else if g := <-c.Callback; g.Result != res {
				t.Errorf("expected %v, got %v", res, g.Result)
			}
		}
	}

	s.MockCallback(gntp.TIMEOUT, gntp.NONE)
	if _, err := c.Notify(new(gntp.Notification)); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Microsecond)
	c.Reset()
	c.Wait()
}

func TestCallbackError(t *testing.T) {
	s := NewServer()
	s.SetPassword(password)
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name
	c.Password = password

	s.MockOK("REGISTER", gntp.NONE)
	_, err := c.Register([]*gntp.Notification{
		{
			Name:    "Name",
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// invalid -CALLBACK response
	s.MockEncryptedResponse(gntp.NONE, func(conn net.Conn, i *gntp.Info) {
		s.OK(conn, i, "NOTIFY")

		io.WriteString(conn, "GNTP/1.0 _ NONE\r\n\r\n")
	})
	if _, err := c.Notify(new(gntp.Notification)); err != nil {
		t.Error(err)
	}
	s.MockEncryptedResponse(gntp.NONE, func(conn net.Conn, i *gntp.Info) {
		s.OK(conn, i, "NOTIFY")

		io.WriteString(conn, "GNTP/1.0 -OK NONE\r\n\r\n")
	})
	if _, err := c.Notify(new(gntp.Notification)); err != nil {
		t.Error(err)
	}
	s.MockEncryptedResponse(gntp.NONE, func(conn net.Conn, i *gntp.Info) {
		s.OK(conn, i, "NOTIFY")

		i.MessageType = "-CALLBACK"

		fmt.Fprintf(conn, "%v\r\n", i)
		io.WriteString(conn, "Application-Name\r\n\r\n")
	})
	if _, err := c.Notify(new(gntp.Notification)); err != nil {
		t.Error(err)
	}
	// invalid -CALLBACK response (encrypted)
	s.MockEncryptedResponse(gntp.AES, func(conn net.Conn, i *gntp.Info) {
		s.OK(conn, i, "NOTIFY")

		i.MessageType = "-CALLBACK"

		fmt.Fprintf(conn, "%v\r\n", i)
	})
	if _, err := c.Notify(new(gntp.Notification)); err != nil {
		t.Error(err)
	}
	s.MockEncryptedResponse(gntp.AES, func(conn net.Conn, i *gntp.Info) {
		s.OK(conn, i, "NOTIFY")

		i.MessageType = "-CALLBACK"

		fmt.Fprintf(conn, "%v\r\n", i)
		bs := i.Cipher().BlockSize()
		src := make([]byte, bs)
		for i := range len(src) {
			src[i] = byte(i % (bs / 2))
		}
		conn.Write(encrypt(i, src))
		io.WriteString(conn, "\r\n\r\n")
	})
	if _, err := c.Notify(new(gntp.Notification)); err != nil {
		t.Error(err)
	}

	c.Wait()
}

func TestRequestError(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name
	c.Password = password

	// connection error
	c.Server = ""
	if _, err := c.Send("_"); err == nil {
		t.Error("expected error")
	}
	// unknown hash algorithm
	c.Server = s.Addr
	c.HashAlgorithm = -1
	c.EncryptionAlgorithm = 0
	if _, err := c.Send("_"); err != gntp.ErrHash {
		t.Errorf("expected ErrHash, got %v", err)
	}
	// unknown encryption algorithm
	c.Server = s.Addr
	c.HashAlgorithm = 0
	c.EncryptionAlgorithm = -1
	if _, err := c.Send("_"); err != gntp.ErrEncryption {
		t.Errorf("expected ErrEncryption, got %v", err)
	}
}

func TestResponse(t *testing.T) {
	s := NewServer()
	defer s.Close()

	c := gntp.New()
	c.Server = s.Addr
	c.Name = name

	// -OK response
	okResp := &gntp.Response{
		Action: "REGISTER",
		Header: make(textproto.MIMEHeader),
	}
	for _, ea := range []gntp.EncryptionAlgorithm{
		gntp.NONE,
		gntp.DES,
		gntp.TDES,
		gntp.AES,
	} {
		if ea != gntp.NONE {
			s.SetPassword(password)
			c.Password = password
		} else {
			s.SetPassword("")
			c.Password = ""
		}
		s.MockOK("REGISTER", ea)
		switch resp, err := c.Register(nil); {
		case err != nil:
			t.Error(err)
		case !reflect.DeepEqual(resp, okResp):
			t.Errorf("expected %#v, got %#v", okResp, resp)
		}
	}
	// -ERROR response
	code := gntp.InvalidRequest
	errResp := gntp.Error{
		Code:        code,
		Description: code.Description(),
		Header:      make(textproto.MIMEHeader),
	}
	s.MockError(code)
	if _, err := c.Register(nil); !reflect.DeepEqual(err, errResp) {
		t.Errorf("expected %#v, got %#v", errResp, err)
	}
	// invalid response
	s.MockResponse(func(net.Conn) {})
	if _, err := c.Register(nil); err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
	s.MockResponse(func(conn net.Conn) {
		io.WriteString(conn, "\r\n")
	})
	if _, err := c.Register(nil); err != gntp.ErrProtocol {
		t.Errorf("expected ErrProtocol, got %v", err)
	}
	s.MockResponse(func(conn net.Conn) {
		io.WriteString(conn, "GNTP/1.0 _ NONE\r\n\r\n")
	})
	if _, err := c.Register(nil); err != gntp.ErrProtocol {
		t.Errorf("expected ErrProtocol, got %v", err)
	}
	s.MockResponse(func(conn net.Conn) {
		io.WriteString(conn, "GNTP/1.0 REGISTER NONE\r\n\r\n")
	})
	if _, err := c.Register(nil); err != gntp.ErrProtocol {
		t.Errorf("expected ErrProtocol, got %v", err)
	}
	// invalid -OK response
	s.MockResponse(func(conn net.Conn) {
		io.WriteString(conn, "GNTP/1.0 -OK NONE\r\n")
		io.WriteString(conn, "Notification-ID\r\n\r\n")
	})
	if _, err := c.Register(nil); err == nil {
		t.Error("expected error")
	}
	// invalid -OK response (encrypted)
	s.MockEncryptedResponse(gntp.AES, func(conn net.Conn, i *gntp.Info) {
		i.MessageType = "-OK"

		fmt.Fprintf(conn, "%v\r\n", i)
	})
	if _, err := c.Register(nil); err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
	s.MockEncryptedResponse(gntp.AES, func(conn net.Conn, i *gntp.Info) {
		i.MessageType = "-OK"

		fmt.Fprintf(conn, "%v\r\n", i)
		bs := i.Cipher().BlockSize()
		src := make([]byte, bs)
		for i := range len(src) {
			src[i] = byte(i % (bs / 2))
		}
		conn.Write(encrypt(i, src))
		io.WriteString(conn, "\r\n\r\n")
	})
	if _, err := c.Register(nil); err != gntp.ErrPKCS7 {
		t.Errorf("expected ErrPKCS7, got %v", err)
	}
	// invalid -ERROR response
	s.MockResponse(func(conn net.Conn) {
		io.WriteString(conn, "GNTP/1.0 -ERROR NONE\r\n")
		io.WriteString(conn, "Error-Code\r\n\r\n")
	})
	if _, err := c.Register(nil); err == nil {
		t.Error("expected error")
	}
	s.MockResponse(func(conn net.Conn) {
		io.WriteString(conn, "GNTP/1.0 -ERROR NONE\r\n")
		io.WriteString(conn, "Error-Code: _\r\n\r\n")
	})
	if _, err := c.Register(nil); err == nil {
		t.Error("expected error")
	}
	// invalid -ERROR response (encrypted)
	s.MockEncryptedResponse(gntp.AES, func(conn net.Conn, i *gntp.Info) {
		i.MessageType = "-ERROR"

		fmt.Fprintf(conn, "%v\r\n", i)
	})
	if _, err := c.Register(nil); err != gntp.ErrProtocol {
		t.Errorf("expected ErrProtocol, got %v", err)
	}
}

func TestInfo(t *testing.T) {
	for _, l := range []string{
		// plain text
		"GNTP/1.0 REGISTER NONE",
		// auth
		"GNTP/1.0 REGISTER NONE MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
		"GNTP/1.0 REGISTER NONE SHA1:926D135D821E07CD720E63FAB2629887E67A3601.0123456789",
		"GNTP/1.0 REGISTER NONE SHA256:CF0D52E2716F54C0EA9D6BAD563F1E1C7C46122BE8BE9FB1A09587D064C723C7.0123456789",
		"GNTP/1.0 REGISTER NONE SHA512:710F213B1F8E97C5BF04089367B4AE08BBDF82285557B4986E3170A3F214165B6320E4C63A8A55A6BD31652FEB9B17B8191B2884AE76D36AFEBF72298B982511.0123456789",
		// encrypt
		"GNTP/1.0 REGISTER DES:0011223344556677 MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
		"GNTP/1.0 REGISTER 3DES:0011223344556677 SHA256:CF0D52E2716F54C0EA9D6BAD563F1E1C7C46122BE8BE9FB1A09587D064C723C7.0123456789",
		"GNTP/1.0 REGISTER AES:00112233445566778899AABBCCDDEEFF SHA256:CF0D52E2716F54C0EA9D6BAD563F1E1C7C46122BE8BE9FB1A09587D064C723C7.0123456789",
	} {
		info, err := gntp.ParseInfo(l, password)
		if err != nil {
			t.Error(err)
		}
		if g, e := info.String(), l; g != e {
			t.Errorf("expected %q, got %q", e, g)
		}
	}
	// error
	for _, l := range []string{
		"",
		// <version>
		"GNTP/0.0",
		"GNTP/0.0 ",
		// <messagetype>
		"GNTP/1.0 _",
		"GNTP/1.0 _ NONE",
		// <encryptionAlgorithmID>
		"GNTP/1.0 REGISTER _",
		"GNTP/1.0 REGISTER AES MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
		"GNTP/1.0 REGISTER NONE:FF MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
		"GNTP/1.0 REGISTER _:FF MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
		// <ivValue>
		"GNTP/1.0 REGISTER AES:_ _",
		"GNTP/1.0 REGISTER DES:FF MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
		// <keyHashAlgorithmID>
		"GNTP/1.0 REGISTER NONE MD5",
		"GNTP/1.0 REGISTER NONE SHA224:D674BB58EDC717D2E44413AB45D8570C4922D6DA732788C166114D87.0123456789",
		// <keyHash>
		"GNTP/1.0 REGISTER NONE MD5:_",
		"GNTP/1.0 REGISTER NONE MD5:_._",
		// <salt>
		"GNTP/1.0 REGISTER NONE MD5:B80A1CD3F719006F932A3FAAC90FEEA5._",
		// <keyHash> verification error
		"GNTP/1.0 REGISTER NONE MD5:B80A1CD3F719006F932A3FAAC90FEEA5.9876543210",
		"GNTP/1.0 REGISTER NONE SHA1:926D135D821E07CD720E63FAB2629887E67A3601.9876543210",
		"GNTP/1.0 REGISTER NONE SHA256:CF0D52E2716F54C0EA9D6BAD563F1E1C7C46122BE8BE9FB1A09587D064C723C7.9876543210",
		"GNTP/1.0 REGISTER NONE SHA512:710F213B1F8E97C5BF04089367B4AE08BBDF82285557B4986E3170A3F214165B6320E4C63A8A55A6BD31652FEB9B17B8191B2884AE76D36AFEBF72298B982511.9876543210",
		// <encryptionAlgorithmID> key length error
		"GNTP/1.0 REGISTER 3DES:FF MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
		"GNTP/1.0 REGISTER AES:FF MD5:B80A1CD3F719006F932A3FAAC90FEEA5.0123456789",
	} {
		if _, err := gntp.ParseInfo(l, password); err == nil {
			t.Error("expected error")
		}
	}
}

func TestDecrypt(t *testing.T) {
	e := []byte("data")
	i := &gntp.Info{
		EncryptionAlgorithm: gntp.AES,
		HashAlgorithm:       gntp.SHA256,
	}

	src := make([]byte, len(e))
	copy(src[:], e[:])
	switch g, err := i.Decrypt(src); {
	case err != nil:
		t.Error(err)
	case !reflect.DeepEqual(g, e):
		t.Errorf("expected %v, got %v", e, g)
	}
	i.SetPassword(password)
	bs := i.Cipher().BlockSize()
	src = bytes.Repeat([]byte{byte(bs - len(e))}, bs)
	copy(src[:], e[:])
	switch g, err := i.Decrypt(encrypt(i, src)); {
	case err != nil:
		t.Error(err)
	case !reflect.DeepEqual(g, e):
		t.Errorf("expected %v, got %v", e, g)
	}
	// invalid PKCS #7 padding
	src = bytes.Repeat([]byte{byte(bs + 1)}, bs)
	if _, err := i.Decrypt(encrypt(i, src)); err != gntp.ErrPKCS7 {
		t.Errorf("expected ErrPKCS7, got %v", err)
	}
	src = make([]byte, bs)
	for i := range len(src) {
		src[i] = byte(i % (bs / 2))
	}
	if _, err := i.Decrypt(encrypt(i, src)); err != gntp.ErrPKCS7 {
		t.Errorf("expected ErrPKCS7, got %v", err)
	}
}

func encrypt(i *gntp.Info, src []byte) []byte {
	dst := make([]byte, len(src))
	cbc := cipher.NewCBCEncrypter(i.Cipher(), i.IV)
	cbc.CryptBlocks(dst, src)
	return dst
}

func TestEncrypt(t *testing.T) {
	e := []byte("data")
	i := &gntp.Info{
		EncryptionAlgorithm: gntp.AES,
		HashAlgorithm:       gntp.SHA256,
	}
	for _, s := range []string{"", password} {
		i.SetPassword(s)
		switch g, err := i.Decrypt(i.Encrypt(e)); {
		case err != nil:
			t.Error(err)
		case !reflect.DeepEqual(g, e):
			t.Errorf("expected %v, got %v", e, g)
		}
	}
}

func TestHashAlgorithm(t *testing.T) {
	for i, e := range []string{
		"MD5",
		"SHA1",
		"SHA256",
		"SHA512",
	} {
		ha := gntp.HashAlgorithm(i)
		if g := ha.String(); g != e {
			t.Errorf("HashAlgorithm.String() = %v, expected %v", g, e)
		}
		if _, err := ha.New(); err != nil {
			t.Errorf("%v: %v", e, err)
		}
	}

	ha := gntp.HashAlgorithm(-1)
	if g, e := ha.String(), "HashAlgorithm(-1)"; g != e {
		t.Errorf("HashAlgorithm.String() = %v, expected %v", g, e)
	}
	if _, err := ha.New(); err == nil {
		t.Error("expected error")
	}
}

func TestEncryptionAlgorithm(t *testing.T) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	io.WriteString(h, password)
	h.Write(salt)
	k := h.Sum(nil)

	for i, e := range []string{
		"NONE",
		"DES",
		"3DES",
		"AES",
	} {
		ea := gntp.EncryptionAlgorithm(i)
		if g := ea.String(); g != e {
			t.Errorf("EncryptionAlgorithm.String() = %v, expected %v", g, e)
		}
		if _, err := ea.New(k); err != nil {
			t.Error(err)
		}
		if _, err := ea.New(nil); ea != gntp.NONE && err == nil {
			t.Error("expected error")
		}
	}

	ea := gntp.EncryptionAlgorithm(-1)
	if g, e := ea.String(), "EncryptionAlgorithm(-1)"; g != e {
		t.Errorf("EncryptionAlgorithm.String() = %v, expected %v", g, e)
	}
	if _, err := ea.New(k); err == nil {
		t.Error("expected error")
	}
}

func TestError(t *testing.T) {
	err := gntp.Error{
		Code:        100,
		Description: "Description",
	}
	if g, e := err.Error(), "Description"; g != e {
		t.Errorf("Error.Error() = %q, expected %q", g, e)
	}

	code := gntp.InternalServerError
	err = gntp.Error{Code: code}
	if g, e := err.Error(), code.Description(); g != e {
		t.Errorf("Error.Error() = %q, expected %q", g, e)
	}
}

func TestResult(t *testing.T) {
	for i, e := range []string{
		"Result(0)",
		"CLICKED",
		"CLOSED",
		"TIMEOUT",
	} {
		res := gntp.Result(i)
		if g := res.String(); g != e {
			t.Errorf("Result.String() = %v, expected %v", g, e)
		}
	}
}
