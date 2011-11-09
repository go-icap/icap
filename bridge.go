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

// A bridge between ICAP and HTTP.
// It allows answering a REQMOD request with an HTTP response generated locally.

package icap

import (
	"log"
	"net/http"
	"time"
)

type bridgedRespWriter struct {
	irw         ResponseWriter // the underlying icap.ResponseWriter
	header      http.Header    // the headers for the HTTP response
	wroteHeader bool           // Have the headers been written yet?
}

func (w *bridgedRespWriter) Header() http.Header {
	return w.header
}

func (w *bridgedRespWriter) Write(p []byte) (n int, err error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.irw.Write(p)
}

func (w *bridgedRespWriter) WriteHeader(code int) {
	if w.wroteHeader {
		log.Print("http: multiple response.WriteHeader calls")
		return
	}

	w.wroteHeader = true

	// Default output is HTML encoded in UTF-8.
	if w.header.Get("Content-Type") == "" {
		w.header.Set("Content-Type", "text/html; charset=utf-8")
	}

	if _, ok := w.header["Date"]; !ok {
		w.Header().Set("Date", time.UTC().Format(http.TimeFormat))
	}

	resp := new(http.Response)
	resp.StatusCode = code
	resp.Header = w.header

	w.irw.WriteHeader(200, resp, true)
}

// Create an http.ResponseWriter that encapsulates its response in an ICAP response.
func NewBridgedResponseWriter(w ResponseWriter) http.ResponseWriter {
	rw := new(bridgedRespWriter)
	rw.header = make(http.Header)
	rw.irw = w

	return rw
}

// Pass use the local HTTP server to generate a response for an ICAP request.
func ServeLocally(w ResponseWriter, req *Request) {
	brw := NewBridgedResponseWriter(w)
	http.DefaultServeMux.ServeHTTP(brw, req.Request)
}
