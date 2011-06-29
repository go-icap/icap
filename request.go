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

// Reading and parsing of ICAP requests.

// Package icap provides an extensible ICAP server.
// (At least it will when it is finished!)
package icap

import (
	"http"
	"net/textproto"
	"io"
	"os"
	"strings"
	"fmt"
	"bufio"
	"strconv"
)

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) String() string { return fmt.Sprintf("%s %q", e.what, e.str) }

// A Request represents a parsed ICAP request.
type Request struct {
	Method string    // REQMOD, RESPMOD, OPTIONS, etc.
	RawURL string    // The URL given in the request.
	URL    *http.URL // Parsed URL.

	// The protocol version.
	Proto string // "ICAP/1.0"

	// The ICAP header.
	Header textproto.MIMEHeader
}

// ReadRequest reads and parses a request from b.
func ReadRequest(b *bufio.Reader) (req *Request, err os.Error) {
	tp := textproto.NewReader(b)
	req = new(Request)

	// Read first line.
	var s string
	s, err = tp.ReadLine()
	if err != nil {
		if err == os.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}

	f := strings.SplitN(s, " ", 3)
	if len(f) < 3 {
		return nil, &badStringError{"malformed ICAP request", s}
	}
	req.Method, req.RawURL, req.Proto = f[0], f[1], f[2]

	req.URL, err = http.ParseRequestURL(req.RawURL)
	if err != nil {
		return nil, err
	}

	req.Header, err = tp.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}

	s = req.Header.Get("Encapsulated")
	if s == "" {
		return nil, os.NewError("missing Encapsulated: header")
	}
	eList := strings.Split(s, ", ")
	var initialOffset, reqHdrLen, respHdrLen int
	var hasBody bool
	var prevKey string
	var prevValue int
	for _, item := range eList {
		eq := strings.Index(item, "=")
		if eq == -1 {
			return nil, &badStringError{"malformed Encapsulated: header", s}
		}
		key := item[:eq]
		value, err := strconv.Atoi(item[eq+1:])
		if err != nil {
			return nil, &badStringError{"malformed Encapsulated: header", s}
		}

		// Calculate the length of the previous section.
		switch prevKey {
		case "":
			initialOffset = value
		case "req-hdr":
			reqHdrLen = value - prevValue
		case "res-hdr":
			respHdrLen = value - prevValue
		case "req-body", "opt-body", "res-body", "null-body":
			return nil, os.NewError(fmt.Sprintf("%s must be the last section", prevKey))
		}

		switch key {
		case "req-hdr", "res-hdr", "null-body":
		case "req-body", "res-body", "opt-body":
			hasBody = true
		default:
			return nil, &badStringError{"invalid key for Encapsulated: header", key}
		}

		prevValue = value
		prevKey = key
	}

	// Read the HTTP headers.
	var rawReqHdr, rawRespHdr []byte
	if initialOffset > 0 {
		junk := make([]byte, initialOffset)
		_, err = io.ReadFull(b, junk)
		if err != nil {
			return nil, err
		}
	}
	if reqHdrLen > 0 {
		rawReqHdr = make([]byte, reqHdrLen)
		_, err = io.ReadFull(b, rawReqHdr)
		if err != nil {
			return nil, err
		}
	}
	if respHdrLen > 0 {
		rawRespHdr = make([]byte, respHdrLen)
		_, err = io.ReadFull(b, rawRespHdr)
		if err != nil {
			return nil, err
		}
	}

	if rawReqHdr != nil {
		fmt.Println("Request header:")
		os.Stdout.Write(rawReqHdr)
	}
	if rawRespHdr != nil {
		fmt.Println("Response header:")
		os.Stdout.Write(rawRespHdr)
	}

	if hasBody {
		fmt.Println("Has a body")
	} else {
		fmt.Println("Body is null")
	}

	return
}
