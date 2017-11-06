//
// go.notify/gntp :: mock_test.go
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

package gntp_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"

	"github.com/hattya/go.notify/gntp"
	"github.com/hattya/go.notify/internal/util"
)

type Server struct {
	Addr string

	l  net.Listener
	wg sync.WaitGroup

	mu       sync.Mutex
	password string
	handlers []func(net.Conn)
	done     chan struct{}
}

func NewServer() *Server {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	s := &Server{
		Addr: l.Addr().String(),
		l:    l,
		done: make(chan struct{}),
	}
	s.wg.Add(1)
	go s.serve()
	return s
}

func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.done:
	default:
		return
	}

	close(s.done)
	s.l.Close()
	s.wg.Wait()
}

func (s *Server) SetPassword(password string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.password = password
}

func (s *Server) Error(conn net.Conn, code gntp.ErrorCode) {
	io.WriteString(conn, "GNTP/1.0 -ERROR NONE\r\n")
	fmt.Fprintf(conn, "Error-Code: %v\r\n", code)
	fmt.Fprintf(conn, "Error-Description: %v\r\n", code.Description())
	io.WriteString(conn, "\r\n")
}

func (s *Server) MockOK(action string) {
	s.MockResponse(func(conn net.Conn) {
		io.WriteString(conn, "GNTP/1.0 -OK NONE\r\n")
		fmt.Fprintf(conn, "Response-Action: %v\r\n", strings.ToUpper(action))
		io.WriteString(conn, "Notification-ID:\r\n\r\n")
	})
}

func (s *Server) MockError(code gntp.ErrorCode) {
	s.MockResponse(func(conn net.Conn) {
		s.Error(conn, code)
	})
}

func (s *Server) MockResponse(handler func(net.Conn)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers = append(s.handlers, handler)
}

func (s *Server) serve() {
	defer s.wg.Done()

	for {
		conn, err := s.l.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
			}
			panic(err)
		}

		s.wg.Add(1)
		go s.handle(conn)

		select {
		case <-s.done:
			return
		default:
		}
	}
}

func (s *Server) handle(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	br := bufio.NewReader(conn)
	// GNTP information
	l, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return
	}
	s.mu.Lock()
	i, err := gntp.ParseInfo(l, s.password)
	s.mu.Unlock()
	if err != nil {
		s.Error(conn, gntp.UnknownProtocol)
		return
	}
	// auth
	if s.password != "" && i.KeyHash == nil {
		s.Error(conn, gntp.NotAuthorized)
		return
	}
	// headers
	blob := 0
	r := textproto.NewReader(br)
	if i.EncryptionAlgorithm != gntp.NONE {
		src, err := util.ReadBytes(br, []byte("\r\n\r\n"))
		if err != nil {
			panic(err)
		}
		b, err := i.Decrypt(src[:len(src)-4])
		if err != nil {
			panic(err)
		}
		blob = s.numBlob(i, textproto.NewReader(bufio.NewReader(bytes.NewReader(b))))
	} else {
		blob = s.numBlob(i, r)
	}
	// identifiers
	for i := 0; i < blob; i++ {
		hdr, err := r.ReadMIMEHeader()
		if err != nil {
			panic(err)
		}
		i, err := strconv.Atoi(hdr.Get("Length"))
		if err != nil {
			panic(err)
		}
		if _, err := br.Discard(i); err != nil {
			panic(err)
		}
		s.crlf(br)
		s.crlf(br)
	}
	s.crlf(br)

	// response
	if 0 < len(s.handlers) {
		s.mu.Lock()
		fn := s.handlers[0]
		s.handlers = s.handlers[1:]
		s.mu.Unlock()
		fn(conn)
	} else {
		s.Error(conn, gntp.InternalServerError)
	}
}

func (s *Server) numBlob(i *gntp.Info, r *textproto.Reader) int {
	blob := make(map[string]struct{})
	find := func(hdr textproto.MIMEHeader) {
		for _, v := range hdr {
			for _, v := range v {
				if strings.HasPrefix(v, "x-growl-resource://") {
					blob[v[19:]] = struct{}{}
				}
			}
		}
	}

	hdr, err := r.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		panic(err)
	}
	find(hdr)
	if i.MessageType == "REGISTER" {
		i, err := strconv.Atoi(hdr.Get("Notifications-Count"))
		if err != nil {
			panic(err)
		}
		for ; 0 < i; i-- {
			hdr, err := r.ReadMIMEHeader()
			if err != nil && err != io.EOF {
				panic(err)
			}
			find(hdr)
		}
	}
	return len(blob)
}

func (s *Server) crlf(r *bufio.Reader) {
	b, err := r.ReadBytes('\n')
	switch {
	case err != nil:
		panic(err)
	case len(b) != 2 || b[0] != '\r':
		panic("expected CRLF")
	}
}

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
