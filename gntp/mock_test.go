//
// go.notify/gntp :: mock_test.go
//
//   Copyright (c) 2017-2025 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
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
	"time"

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
	select {
	case <-s.done:
		s.mu.Unlock()
		return
	default:
		close(s.done)
	}
	s.mu.Unlock()

	s.l.Close()
	s.wg.Wait()
}

func (s *Server) SetPassword(password string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.password = password
}

func (s *Server) OK(conn net.Conn, i *gntp.Info, action string) {
	i.MessageType = "-OK"

	fmt.Fprintf(conn, "%v\r\n", i)
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "Response-Action: %v\r\n", strings.ToUpper(action))
	b.WriteString("Notification-ID:\r\n")
	if i.EncryptionAlgorithm != gntp.NONE {
		conn.Write(i.Encrypt(b.Bytes()))
		io.WriteString(conn, "\r\n\r\n")
	} else {
		conn.Write(b.Bytes())
		io.WriteString(conn, "\r\n")
	}
}

func (s *Server) Error(conn net.Conn, code gntp.ErrorCode) {
	io.WriteString(conn, "GNTP/1.0 -ERROR NONE\r\n")
	fmt.Fprintf(conn, "Error-Code: %v\r\n", code)
	fmt.Fprintf(conn, "Error-Description: %v\r\n", code.Description())
	io.WriteString(conn, "\r\n")
}

func (s *Server) MockOK(action string, ea gntp.EncryptionAlgorithm) {
	s.MockEncryptedResponse(ea, func(conn net.Conn, i *gntp.Info) {
		s.OK(conn, i, action)
	})
}

func (s *Server) MockError(code gntp.ErrorCode) {
	s.MockResponse(func(conn net.Conn) {
		s.Error(conn, code)
	})
}

func (s *Server) MockCallback(res gntp.Result, ea gntp.EncryptionAlgorithm) {
	s.MockEncryptedResponse(ea, func(conn net.Conn, i *gntp.Info) {
		s.OK(conn, i, "NOTIFY")
		// callback
		i.MessageType = "-CALLBACK"

		fmt.Fprintf(conn, "%v\r\n", i)
		b := new(bytes.Buffer)
		b.WriteString("Application-Name:\r\n")
		b.WriteString("Notification-ID:\r\n")
		fmt.Fprintf(b, "Notification-Callback-Result: %v\r\n", res)
		fmt.Fprintf(b, "Notification-Callback-Timestamp: %v\r\n", time.Now().Format(gntp.RFC3339))
		b.WriteString("Notification-Callback-Context: context\r\n")
		b.WriteString("Notification-Callback-Context-Type: context-type\r\n")
		if i.EncryptionAlgorithm != gntp.NONE {
			conn.Write(i.Encrypt(b.Bytes()))
			io.WriteString(conn, "\r\n\r\n")
		} else {
			conn.Write(b.Bytes())
			io.WriteString(conn, "\r\n")
		}
	})
}

func (s *Server) MockEncryptedResponse(ea gntp.EncryptionAlgorithm, handler func(net.Conn, *gntp.Info)) {
	s.MockResponse(func(conn net.Conn) {
		i := &gntp.Info{
			Version:             "1.0",
			HashAlgorithm:       gntp.SHA256,
			EncryptionAlgorithm: ea,
		}
		s.mu.Lock()
		pwd := s.password
		s.mu.Unlock()
		if err := i.SetPassword(pwd); err != nil {
			panic(err)
		}
		handler(conn, i)
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
				panic(err)
			}
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
	pwd := s.password
	s.mu.Unlock()
	i, err := gntp.ParseInfo(l, pwd)
	if err != nil {
		s.Error(conn, gntp.UnknownProtocol)
		return
	}
	// auth
	if pwd != "" && i.KeyHash == nil {
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
	for range blob {
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
	s.mu.Lock()
	if len(s.handlers) != 0 {
		defer s.handlers[0](conn)
		s.handlers = s.handlers[1:]
	} else {
		s.Error(conn, gntp.InternalServerError)
	}
	s.mu.Unlock()
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
		for ; i > 0; i-- {
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
