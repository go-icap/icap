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

// Responding to ICAP requests.

package icap

import (
	"bytes"
	"fmt"
	"http"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"url"
)

type ResponseWriter interface {
	// Header returns the header map that will be sent by WriteHeader.
	// Changing the header after a call to WriteHeader (or Write) has
	// no effect.
	Header() http.Header

	// Write writes the data to the connection as part of an ICAP reply.
	// If WriteHeader has not yet been called, Write calls WriteHeader(http.StatusOK, nil)
	// before writing the data.
	Write([]byte) (int, os.Error)

	// WriteHeader sends an ICAP response header with status code.
	// Then it sends an HTTP header if httpMessage is not nil.
	// httpMessage may be an *http.Request or an *http.Response.
	// hasBody should be true if there will be calls to Write(), generating a message body.
	WriteHeader(code int, httpMessage interface{}, hasBody bool)
}

type respWriter struct {
	conn        *conn          // information on the connection
	req         *Request       // the request that is being responded to
	header      http.Header    // the ICAP header to write for the response
	wroteHeader bool           // true if the headers have already been written
	cw          io.WriteCloser // the chunked writer used to write the body
}

func (w *respWriter) Header() http.Header {
	return w.header
}

func (w *respWriter) Write(p []byte) (n int, err os.Error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK, nil, true)
	}

	if w.cw == nil {
		return 0, os.NewError("called Write() on an icap.ResponseWriter that should not have a body")
	}
	return w.cw.Write(p)
}

func (w *respWriter) WriteHeader(code int, httpMessage interface{}, hasBody bool) {
	if w.wroteHeader {
		log.Println("Called WriteHeader twice on the same connection")
		return
	}

	// Make the HTTP header and the Encapsulated: header.
	var header []byte
	var encap string
	var err os.Error

	switch msg := httpMessage.(type) {
	case *http.Request:
		header, err = httpRequestHeader(msg)
		if err != nil {
			break
		}
		if hasBody {
			encap = fmt.Sprintf("req-hdr=0, req-body=%d", len(header))
		} else {
			encap = fmt.Sprintf("req-hdr=0, null-body=%d", len(header))
		}

	case *http.Response:
		header, err = httpResponseHeader(msg)
		if err != nil {
			break
		}
		if hasBody {
			encap = fmt.Sprintf("res-hdr=0, res-body=%d", len(header))
		} else {
			encap = fmt.Sprintf("res-hdr=0, null-body=%d", len(header))
		}
	}

	if encap == "" {
		if hasBody {
			method := w.req.Method
			if len(method) > 3 {
				method = method[0:3]
			}
			method = strings.ToLower(method)
			encap = fmt.Sprintf("%s-body=0", method)
		} else {
			encap = "null-body=0"
		}
	}

	w.header.Set("Encapsulated", encap)
	if _, ok := w.header["Date"]; !ok {
		w.Header().Set("Date", time.UTC().Format(http.TimeFormat))
	}

	w.header.Set("Connection", "close")

	bw := w.conn.buf.Writer
	status := StatusText(code)
	if status == "" {
		status = fmt.Sprintf("status code %d", code)
	}
	fmt.Fprintf(bw, "ICAP/1.0 %d %s\r\n", code, status)
	w.header.Write(bw)
	io.WriteString(bw, "\r\n")

	if header != nil {
		bw.Write(header)
	}

	w.wroteHeader = true

	if hasBody {
		w.cw = http.NewChunkedWriter(w.conn.buf.Writer)
	}
}

func (w *respWriter) finishRequest() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK, nil, false)
	}

	if w.cw != nil {
		w.cw.Close()
		w.cw = nil
		io.WriteString(w.conn.buf, "\r\n")
	}

	w.conn.buf.Flush()
}

// httpRequestHeader returns the headers for an HTTP request
// as a slice of bytes in a form suitable for including in an ICAP message.
func httpRequestHeader(req *http.Request) (hdr []byte, err os.Error) {
	buf := new(bytes.Buffer)

	if req.URL == nil {
		req.URL, err = url.Parse(req.RawURL)
		if err != nil {
			return nil, os.NewError("icap: httpRequestHeader called on Request with no URL")
		}
	}

	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	req.Header.Set("Host", host)

	uri := req.URL.String()

	fmt.Fprintf(buf, "%s %s %s\r\n", valueOrDefault(req.Method, "GET"), uri, valueOrDefault(req.Proto, "HTTP/1.1"))
	req.Header.WriteSubset(buf, map[string]bool{
		"Transfer-Encoding": true,
		"Content-Length":    true,
	})
	io.WriteString(buf, "\r\n")

	return buf.Bytes(), nil
}

// httpResponseHeader returns the headers for an HTTP response
// as a slice of bytes.
func httpResponseHeader(resp *http.Response) (hdr []byte, err os.Error) {
	buf := new(bytes.Buffer)

	// Status line
	text := resp.Status
	if text == "" {
		text = http.StatusText(resp.StatusCode)
		if text == "" {
			text = "status code " + strconv.Itoa(resp.StatusCode)
		}
	}
	proto := resp.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	fmt.Fprintf(buf, "%s %d %s\r\n", proto, resp.StatusCode, text)
	resp.Header.WriteSubset(buf, map[string]bool{
		"Transfer-Encoding": true,
		"Content-Length":    true,
	})
	io.WriteString(buf, "\r\n")

	return buf.Bytes(), nil
}

// Return value if nonempty, def otherwise.
func valueOrDefault(value, def string) string {
	if value != "" {
		return value
	}
	return def
}
