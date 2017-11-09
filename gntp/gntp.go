//
// go.notify/gntp :: gntp.go
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

//go:generate stringer -type HashAlgorithm,Result -output ${GOPACKAGE}_string.go

// Package gntp implements the Growl Notification Transport Protocol version
// 1.0.
//
// See http://www.growlforwindows.com/gfw/help/gntp.aspx for details.
package gntp

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"net"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hattya/go.notify/internal/util"
)

var (
	ErrProtocol   = errors.New("go.notify: protocol error")
	ErrHash       = errors.New("go.notify: unknown hash algorithm")
	ErrEncryption = errors.New("go.notify: unknown encryption algorithm")
	ErrKeyLength  = errors.New("go.notify: key length is too short")
	ErrPassword   = errors.New("go.notify: incorrect password")
	ErrPKCS7      = errors.New("go.notify: invalid PKCS #7 padding")
)

const rfc3339 = "2006-01-02 15:04:05Z"

// Client is a GNTP client.
type Client struct {
	Server              string
	Name                string
	Icon                Icon
	Password            string
	HashAlgorithm       HashAlgorithm
	EncryptionAlgorithm EncryptionAlgorithm

	// Custom Headers and App-Specific Headers
	Header map[string]interface{}

	Callback chan *Callback
	wg       sync.WaitGroup // for testing

	mu     sync.Mutex
	cb     map[net.Conn]struct{}
	ctx    context.Context
	cancel context.CancelFunc
}

// New returns a new Client.
func New() *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		Server:   "localhost:23053",
		Header:   make(map[string]interface{}),
		Callback: make(chan *Callback),
		cb:       make(map[net.Conn]struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Reset closes connections that are waiting for socket callback.
func (c *Client) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for conn := range c.cb {
		conn.Close()
	}
	c.cancel()

	c.ctx, c.cancel = context.WithCancel(context.Background())
}

// Register sends a REGISTER request to the server.
//
// A REGISTER request only uses the Name, DisplayName, Enabled, and Icon
// fields of the Notification.
func (c *Client) Register(n []*Notification) (*Response, error) {
	b := c.buffer()
	b.Header("Application-Name", c.Name)
	switch icon, err := b.Icon(c.Icon); {
	case err != nil:
		return nil, err
	case icon != "":
		b.Header("Application-Icon", icon)
	}
	b.Header("Notifications-Count", len(n))
	for k, v := range c.Header {
		switch id, err := b.Resource(v); {
		case err != nil:
			return nil, err
		case id != "":
			v = id
		}
		b.Header(textproto.CanonicalMIMEHeaderKey(k), v)
	}
	for _, n := range n {
		b.CRLF()
		b.Header("Notification-Name", n.Name)
		if n.DisplayName != "" {
			b.Header("Notification-Display-Name", n.DisplayName)
		}
		if n.Enabled {
			b.Header("Notification-Enabled", "True")
		}
		switch icon, err := b.Icon(n.Icon); {
		case err != nil:
			return nil, err
		case icon != "":
			b.Header("Notification-Icon", icon)
		}
	}
	return c.send("REGISTER", b)
}

// Notify sends a NOTIFY request to the server.
//
// A NOTIFY request does not use the DisplayName and Enabled fields of the
// Notification.
func (c *Client) Notify(n *Notification) (*Response, error) {
	b := c.buffer()
	b.Header("Application-Name", c.Name)
	b.Header("Notification-Name", n.Name)
	if n.ID != "" {
		b.Header("Notification-ID", n.ID)
	}
	b.Header("Notification-Title", n.Title)
	b.Header("Notification-Text", n.Text)
	if n.Sticky {
		b.Header("Notification-Sticky", "True")
	}
	if n.Priority != 0 {
		b.Header("Notification-Priority", n.Priority)
	}
	switch icon, err := b.Icon(n.Icon); {
	case err != nil:
		return nil, err
	case icon != "":
		b.Header("Notification-Icon", icon)
	}
	if n.CoalescingID != "" {
		b.Header("Notification-Coalescing-ID", n.CoalescingID)
	}
	if n.CallbackContext != "" {
		b.Header("Notification-Callback-Context", n.CallbackContext)
		b.Header("Notification-Callback-Context-Type", n.CallbackContextType)
	}
	if n.CallbackTarget != "" {
		b.Header("Notification-Callback-Target", n.CallbackTarget)
	}
	for k, v := range c.Header {
		switch id, err := b.Resource(v); {
		case err != nil:
			return nil, err
		case id != "":
			v = id
		}
		b.Header(textproto.CanonicalMIMEHeaderKey(k), v)
	}
	return c.send("NOTIFY", b)
}

func (c *Client) buffer() *buffer {
	return &buffer{
		c:    c,
		list: make(map[string][]byte),
	}
}

func (c *Client) send(mt string, b *buffer) (resp *Response, err error) {
	conn, err := net.Dial("tcp", c.Server)
	if err != nil {
		return
	}
	defer func() {
		if err != nil || mt != "NOTIFY" {
			conn.Close()
		}
	}()

	i := &Info{
		Version:             "1.0",
		MessageType:         mt,
		HashAlgorithm:       c.HashAlgorithm,
		EncryptionAlgorithm: c.EncryptionAlgorithm,
	}
	if err = i.SetPassword(c.Password); err != nil {
		return
	}
	io.WriteString(conn, i.String())
	io.WriteString(conn, "\r\n")
	if c.EncryptionAlgorithm != NONE {
		conn.Write(i.Encrypt(b.Bytes()))
		io.WriteString(conn, "\r\n\r\n")
	} else {
		conn.Write(b.Bytes())
		io.WriteString(conn, "\r\n")
	}
	for id, data := range b.list {
		if c.EncryptionAlgorithm != NONE {
			data = i.Encrypt(data)
		}
		fmt.Fprintf(conn, "Identifier: %v\r\n", id)
		fmt.Fprintf(conn, "Length: %v\r\n\r\n", len(data))
		conn.Write(data)
		io.WriteString(conn, "\r\n\r\n")
	}
	io.WriteString(conn, "\r\n")

	// response
	br := bufio.NewReader(conn)
	r := textproto.NewReader(br)
	l, err := r.ReadLine()
	if err != nil {
		return
	}
	i, err = ParseInfo(l, c.Password)
	if err != nil {
		return
	}
	var hdr textproto.MIMEHeader
	switch i.MessageType {
	case "-OK":
		if i.EncryptionAlgorithm != NONE {
			var b []byte
			b, err = util.ReadBytes(br, []byte("\r\n\r\n"))
			if err != nil {
				break
			}
			b, err = i.Decrypt(b[:len(b)-4])
			if err != nil {
				break
			}
			r = textproto.NewReader(bufio.NewReader(bytes.NewReader(b)))
		}
		hdr, err = r.ReadMIMEHeader()
		if err != nil {
			if err != io.EOF {
				break
			}
			err = nil
		}
		resp = &Response{
			Action: hdr.Get("Response-Action"),
			ID:     hdr.Get("Notification-ID"),
			Header: hdr,
		}
		hdr.Del("Response-Action")
		hdr.Del("Notification-ID")
	case "-ERROR":
		if i.EncryptionAlgorithm != NONE {
			err = ErrProtocol
			break
		}
		hdr, err = r.ReadMIMEHeader()
		if err != nil && err != io.EOF {
			break
		}
		var code int
		code, err = strconv.Atoi(hdr.Get("Error-Code"))
		if err != nil {
			break
		}
		err = Error{
			Code:        ErrorCode(code),
			Description: hdr.Get("Error-Description"),
			Header:      hdr,
		}
		hdr.Del("Error-Code")
		hdr.Del("Error-Description")
	default:
		err = ErrProtocol
	}
	// socket callback
	if err == nil && mt == "NOTIFY" {
		c.wg.Add(1)
		c.mu.Lock()
		c.cb[conn] = struct{}{}
		go c.callback(c.ctx, conn, br)
		c.mu.Unlock()
	}
	return
}

func (c *Client) callback(ctx context.Context, conn net.Conn, br *bufio.Reader) {
	defer c.wg.Done()
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		conn.Close()
		delete(c.cb, conn)
	}()

	r := textproto.NewReader(br)
	l, err := r.ReadLine()
	if err != nil {
		return
	}
	i, err := ParseInfo(l, c.Password)
	switch {
	case err != nil:
		return
	case i.MessageType != "-CALLBACK":
		return
	case i.EncryptionAlgorithm != NONE:
		b, err := util.ReadBytes(br, []byte("\r\n\r\n"))
		if err != nil {
			return
		}
		b, err = i.Decrypt(b[:len(b)-4])
		if err != nil {
			return
		}
		r = textproto.NewReader(bufio.NewReader(bytes.NewBuffer(b)))
	}
	hdr, err := r.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		return
	}
	cb := &Callback{
		Name:        hdr.Get("Application-Name"),
		ID:          hdr.Get("Notification-ID"),
		Context:     hdr.Get("Notification-Callback-Context"),
		ContextType: hdr.Get("Notification-Callback-Context-Type"),
		Header:      hdr,
	}
	hdr.Del("Application-Name")
	hdr.Del("Notification-ID")
	hdr.Del("Notification-Callback-Context")
	hdr.Del("Notification-Callback-Context-Type")
	if cb.Name == "" {
		cb.Name = c.Name
	}
	switch strings.ToUpper(hdr.Get("Notification-Callback-Result")) {
	case "CLICKED", "CLICK":
		cb.Result = CLICKED
	case "CLOSED", "CLOSE":
		cb.Result = CLOSED
	case "TIMEDOUT", "TIMEOUT":
		cb.Result = TIMEOUT
	}
	if cb.Result != 0 {
		hdr.Del("Notification-Callback-Result")
	}
	cb.Timestamp, err = time.Parse(rfc3339, hdr.Get("Notification-Callback-Timestamp"))
	if err == nil {
		hdr.Del("Notification-Callback-Timestamp")
	}

	select {
	case c.Callback <- cb:
	case <-ctx.Done():
	}
}

// Icon represents an icon and which supports following types:
//   - string
//   - []byte
//   - image.Image
//   - io.Reader
type Icon interface{}

// HashAlgorithm represents a hash algorithm of the GNTP protocol.
type HashAlgorithm int

// List of hash algorithms for the GNTP protocol.
const (
	MD5 HashAlgorithm = iota
	SHA1
	SHA256
	SHA512
)

// New returns a new hash.Hash.
func (ha HashAlgorithm) New() (h hash.Hash, err error) {
	switch ha {
	case MD5:
		h = md5.New()
	case SHA1:
		h = sha1.New()
	case SHA256:
		h = sha256.New()
	case SHA512:
		h = sha512.New()
	default:
		err = ErrHash
	}
	return
}

// EncryptionAlgorithm represents an encryption algorithm of the GNTP protocol.
type EncryptionAlgorithm int

// List of encryption algorithms for the GNTP protocol.
const (
	NONE EncryptionAlgorithm = iota
	DES
	TDES // 3DES
	AES
)

// New returns a new cipher.Block for encryption and the IV.
func (ea EncryptionAlgorithm) New(key []byte) (cipher.Block, error) {
	var newCipher func([]byte) (cipher.Block, error)
	var n int
	switch ea {
	case NONE:
		return nil, nil
	case DES:
		newCipher = des.NewCipher
		n = 8
	case TDES:
		newCipher = des.NewTripleDESCipher
		n = 24
	case AES:
		newCipher = aes.NewCipher
		n = 24
	default:
		return nil, ErrEncryption
	}
	if len(key) < n {
		return nil, ErrKeyLength
	}
	return newCipher(key[:n])
}

func (ea EncryptionAlgorithm) String() string {
	switch ea {
	case NONE:
		return "NONE"
	case DES:
		return "DES"
	case TDES:
		return "3DES"
	case AES:
		return "AES"
	}
	return fmt.Sprintf("EncryptionAlgorithm(%d)", ea)
}

// Notification represents a notification.
type Notification struct {
	Name                string
	DisplayName         string
	Enabled             bool
	ID                  string
	Title               string
	Text                string
	Sticky              bool
	Priority            int
	Icon                Icon
	CoalescingID        string
	CallbackContext     string
	CallbackContextType string
	CallbackTarget      string
}

var sanitizer = strings.NewReplacer(
	"\r\n", "\n",
	"\r", " ",
)

type buffer struct {
	bytes.Buffer

	c    *Client
	list map[string][]byte
}

func (b *buffer) CRLF() {
	b.WriteString("\r\n")
}

func (b *buffer) Header(key string, value interface{}) {
	if s, ok := value.(string); ok {
		value = sanitizer.Replace(s)
	}
	fmt.Fprintf(b, "%v: %v\r\n", key, value)
}

func (b *buffer) Icon(value interface{}) (id string, err error) {
	switch v := value.(type) {
	case nil:
	case string:
		// <url>
		id = v
	case []byte:
		return b.uniqueid(v)
	case image.Image:
		v, err = util.Convert(v)
		if err != nil {
			return
		}
		w := new(bytes.Buffer)
		if err = png.Encode(w, v); err != nil {
			return
		}
		return b.uniqueid(w.Bytes())
	case io.Reader:
		var data []byte
		data, err = ioutil.ReadAll(v)
		if err != nil {
			return
		}
		return b.uniqueid(data)
	default:
		err = fmt.Errorf("unsupported icon: %T", value)
	}
	return
}

func (b *buffer) Resource(value interface{}) (string, error) {
	switch v := value.(type) {
	case []byte:
		return b.uniqueid(v)
	case io.Reader:
		data, err := ioutil.ReadAll(v)
		if err != nil {
			return "", err
		}
		return b.uniqueid(data)
	}
	return "", nil
}

func (b *buffer) uniqueid(data []byte) (id string, err error) {
	h, err := b.c.HashAlgorithm.New()
	if err != nil {
		return
	}
	h.Write(data)
	id = fmt.Sprintf("x-growl-resource://%X", h.Sum(nil))
	b.list[id[19:]] = data
	return
}

// Info represents a GNTP information line.
type Info struct {
	Version             string
	MessageType         string
	EncryptionAlgorithm EncryptionAlgorithm
	IV                  []byte
	HashAlgorithm       HashAlgorithm
	KeyHash             []byte
	Salt                []byte

	cipher cipher.Block
}

// ParseInfo parses a GNTP information line.
func ParseInfo(l, password string) (i *Info, err error) {
	var x int
	l = strings.ToUpper(l)
	if !strings.HasPrefix(l, "GNTP/") {
		goto Error
	}
	i = new(Info)
	// <version>
	l = l[5:]
	x = strings.IndexRune(l, ' ')
	if x == -1 {
		goto Error
	}
	i.Version = l[:x]
	if i.Version != "1.0" {
		goto Error
	}
	// <messagetype>
	l = strings.TrimSpace(l[x:])
	x = strings.IndexRune(l, ' ')
	if x == -1 {
		goto Error
	}
	i.MessageType = l[:x]
	switch i.MessageType {
	case "REGISTER", "NOTIFY":
	case "-OK", "-ERROR", "-CALLBACK":
	default:
		goto Error
	}
	// <encryptionAlgorithmID>
	l = strings.TrimSpace(l[x:])
	x = strings.IndexRune(l, ' ')
	if x == -1 {
		if l != "NONE" {
			goto Error
		}
	} else {
		eaID := l[:x]
		// <ivValue>
		l = strings.TrimSpace(l[x:])
		x = strings.IndexRune(eaID, ':')
		if x == -1 {
			if eaID != "NONE" {
				goto Error
			}
		} else {
			i.IV, err = hex.DecodeString(eaID[x+1:])
			if err != nil {
				goto Error
			}
			switch eaID[:x] {
			case "NONE":
				goto Error
			case "DES":
				i.EncryptionAlgorithm = DES
			case "3DES":
				i.EncryptionAlgorithm = TDES
			case "AES":
				i.EncryptionAlgorithm = AES
			default:
				i.EncryptionAlgorithm = -1
			}
		}
		// <keyHashAlgorithmID>
		if l != "" {
			x = strings.IndexRune(l, ':')
			if x == -1 {
				goto Error
			}
			switch l[:x] {
			case "MD5":
				i.HashAlgorithm = MD5
			case "SHA1":
				i.HashAlgorithm = SHA1
			case "SHA256":
				i.HashAlgorithm = SHA256
			case "SHA512":
				i.HashAlgorithm = SHA512
			default:
				i.HashAlgorithm = -1
			}
			// <keyHash>
			l = l[x+1:]
			x = strings.IndexRune(l, '.')
			if x == -1 {
				goto Error
			}
			kh, err := hex.DecodeString(l[:x])
			if err != nil {
				goto Error
			}
			// <salt>
			i.Salt, err = hex.DecodeString(l[x+1:])
			if err != nil {
				goto Error
			}
			// verify <keyHash>
			h, err := i.HashAlgorithm.New()
			if err != nil {
				return nil, err
			}
			io.WriteString(h, password)
			h.Write(i.Salt)
			k := h.Sum(nil)
			h.Reset()
			h.Write(k)
			if !reflect.DeepEqual(h.Sum(nil), kh) {
				return nil, ErrPassword
			}
			i.KeyHash = kh
			// verify <ivValue>
			if i.EncryptionAlgorithm != NONE {
				i.cipher, err = i.EncryptionAlgorithm.New(k)
				switch {
				case err != nil:
					return nil, err
				case len(i.IV) != i.cipher.BlockSize():
					goto Error
				}
			}
		}
	}
	return
Error:
	return nil, ErrProtocol
}

// Decrypt decrypts the specified data and removes the PKCS #7 padding.
func (i *Info) Decrypt(data []byte) ([]byte, error) {
	if i.cipher == nil {
		return data, nil
	}
	dst := make([]byte, len(data))
	cbc := cipher.NewCBCDecrypter(i.cipher, i.IV)
	cbc.CryptBlocks(dst, data)
	// PKCS #7 padding
	v := dst[len(dst)-1]
	n := len(dst) - int(v)
	if n < 0 {
		return nil, ErrPKCS7
	}
	for i := n; i < len(dst); i++ {
		if dst[i] != v {
			return nil, ErrPKCS7
		}
	}
	return dst[:n], nil
}

// Encrypt encrypts the specified data with the PKCS #7 padding.
func (i *Info) Encrypt(data []byte) []byte {
	if i.cipher == nil {
		return data
	}
	bs := i.cipher.BlockSize()
	src := make([]byte, int(len(data)/bs)*bs+bs)
	copy(src[:], data[:])
	for i := len(data); i < len(src); i++ {
		src[i] = byte(len(src) - len(data))
	}
	dst := make([]byte, len(src))
	cbc := cipher.NewCBCEncrypter(i.cipher, i.IV)
	cbc.CryptBlocks(dst, src)
	return dst
}

// SetPassword updates the IV, KeyHash, and Salt based on the specified
// password. Their resulting values are dependent on the EncryptionAlgorithm
// and HashAlgorithm.
func (i *Info) SetPassword(password string) (err error) {
	if password == "" {
		i.IV = nil
		i.KeyHash = nil
		i.Salt = nil
		i.cipher = nil
	} else {
		// salt
		if len(i.Salt) == 0 {
			i.Salt = make([]byte, 16)
			if _, err = rand.Read(i.Salt); err != nil {
				return
			}
		}
		// key
		var h hash.Hash
		h, err = i.HashAlgorithm.New()
		if err != nil {
			return
		}
		io.WriteString(h, password)
		h.Write(i.Salt)
		k := h.Sum(nil)
		// key hash
		h.Reset()
		h.Write(k)
		i.KeyHash = h.Sum(nil)

		if i.EncryptionAlgorithm != NONE {
			i.cipher, err = i.EncryptionAlgorithm.New(k)
			if err != nil {
				return
			}
			// iv
			if len(i.IV) != i.cipher.BlockSize() {
				i.IV = make([]byte, i.cipher.BlockSize())
				if _, err = rand.Read(i.IV); err != nil {
					return
				}
			}
		}
	}
	return
}

func (i *Info) String() string {
	switch {
	case i.EncryptionAlgorithm != NONE:
		// encrypt
		return fmt.Sprintf("GNTP/1.0 %v %v:%X %v:%X.%X", i.MessageType, i.EncryptionAlgorithm, i.IV, i.HashAlgorithm, i.KeyHash, i.Salt)
	case len(i.KeyHash) != 0:
		// auth
		return fmt.Sprintf("GNTP/1.0 %v %v %v:%X.%X", i.MessageType, i.EncryptionAlgorithm, i.HashAlgorithm, i.KeyHash, i.Salt)
	default:
		// plain text
		return fmt.Sprintf("GNTP/1.0 %v %v", i.MessageType, i.EncryptionAlgorithm)
	}
}

// Response represents a GNTP response.
type Response struct {
	Action string
	ID     string
	Header textproto.MIMEHeader
}

// Callback represents a GNTP callback
type Callback struct {
	Name        string
	ID          string
	Result      Result
	Timestamp   time.Time
	Context     string
	ContextType string
	Header      textproto.MIMEHeader
}

// Result represents a result of the GNTP callback.
type Result int

// List of results for the GNTP callback.
const (
	CLICKED Result = iota + 1
	CLOSED
	TIMEOUT
)
