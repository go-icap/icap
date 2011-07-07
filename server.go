/*
Copyright © 2011, Andy Balholm
All rights reserved.

Based in part on the http package in the Go standard library (© 2009, the Go Authors).

Redistribution and use in source and binary forms, with or without modification, 
are permitted provided that the following conditions are met:

• Redistributions of source code must retain the above copyright notice, 
this list of conditions and the following disclaimer.

• Redistributions in binary form must reproduce the above copyright notice, 
this list of conditions and the following disclaimer 
in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, 
INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. 
IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, 
OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; 
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, 
WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY 
OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Network connections and request dispatch for the ICAP server.

package icap

import (
	"bufio"
	"bytes"
	"fmt"
	"http"
	"log"
	"net"
	"os"
	"runtime/debug"
)

// Objects implementing the Handler interface can be registered
// to serve ICAP requests.
//
// ServeICAP should write reply headers and data to the ResponseWriter
// and then return.
type Handler interface {
	ServeICAP(ResponseWriter, *Request)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as ICAP handlers.  If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler object that calls f.
type HandlerFunc func(ResponseWriter, *Request)

// ServeICAP calls f(w, r).
func (f HandlerFunc) ServeICAP(w ResponseWriter, r *Request) {
	f(w, r)
}

// A conn represents the server side of an ICAP connection.
type conn struct {
	remoteAddr string            // network address of remote side
	handler    Handler           // request handler
	rwc        net.Conn          // i/o connection
	buf        *bufio.ReadWriter // buffered rwc
}

// Create new connection from rwc.
func newConn(rwc net.Conn, handler Handler) (c *conn, err os.Error) {
	c = new(conn)
	c.remoteAddr = rwc.RemoteAddr().String()
	c.handler = handler
	c.rwc = rwc
	br := bufio.NewReader(rwc)
	bw := bufio.NewWriter(rwc)
	c.buf = bufio.NewReadWriter(br, bw)

	return c, nil
}

// Read next request from connection.
func (c *conn) readRequest() (w *respWriter, err os.Error) {
	var req *Request
	if req, err = ReadRequest(c.buf.Reader); err != nil {
		return nil, err
	}

	req.RemoteAddr = c.remoteAddr

	w = new(respWriter)
	w.conn = c
	w.req = req
	w.header = make(http.Header)
	return w, nil
}

// Close the connection.
func (c *conn) close() {
	if c.buf != nil {
		c.buf.Flush()
		c.buf = nil
	}
	if c.rwc != nil {
		c.rwc.Close()
		c.rwc = nil
	}
}

// Serve a new connection.
func (c *conn) serve() {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		c.rwc.Close()

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "icap: panic serving %v: %v\n", c.remoteAddr, err)
		buf.Write(debug.Stack())
		log.Print(buf.String())
	}()

	for {
		w, err := c.readRequest()
		if err != nil {
			break
			panic(fmt.Errorf("error while reading request: %v", err))
		}

		c.handler.ServeICAP(w, w.req)
		w.finishRequest()
	}

	c.close()
}

// A Server defines parameters for running an ICAP server.
type Server struct {
	Addr         string  // TCP address to listen on, ":1344" if empty
	Handler      Handler // handler to invoke
	ReadTimeout  int64   // the net.Conn.SetReadTimeout value for new connections
	WriteTimeout int64   // the net.Conn.SetWriteTimeout value for new connections
}

// ListenAndServe listens on the TCP network address srv.Addr and then
// calls Serve to handle requests on incoming connections.  If
// srv.Addr is blank, ":1344" is used.
func (srv *Server) ListenAndServe() os.Error {
	addr := srv.Addr
	if addr == "" {
		addr = ":1344"
	}
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return e
	}
	return srv.Serve(l)
}

// Serve accepts incoming connections on the Listener l, creating a
// new service thread for each.  The service threads read requests and
// then call srv.Handler to reply to them.
func (srv *Server) Serve(l net.Listener) os.Error {
	defer l.Close()
	handler := srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}

	for {
		rw, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				log.Printf("icap: Accept error: %v", e)
				continue
			}
			return e
		}
		if srv.ReadTimeout != 0 {
			rw.SetReadTimeout(srv.ReadTimeout)
		}
		if srv.WriteTimeout != 0 {
			rw.SetWriteTimeout(srv.WriteTimeout)
		}
		c, err := newConn(rw, handler)
		if err != nil {
			continue
		}
		go c.serve()
	}
	panic("not reached")
}

// Serve accepts incoming ICAP connections on the listener l,
// creating a new service thread for each.  The service threads
// read requests and then call handler to reply to them.
func Serve(l net.Listener, handler Handler) os.Error {
	srv := &Server{Handler: handler}
	return srv.Serve(l)
}

// ListenAndServe listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.
func ListenAndServe(addr string, handler Handler) os.Error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}
