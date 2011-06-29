package icap

import (
	"testing"
	"bufio"
	"strings"
	"io/ioutil"
)

func checkString(description, is, shouldBe string, t *testing.T) {
	if is != shouldBe {
		t.Fatalf("%s is %s (should be %s)", description, is, shouldBe)
	}
}

func TestParserREQMOD(t *testing.T) {
	buf := strings.NewReader(
		"REQMOD icap://icap-server.net/server?arg=87 ICAP/1.0\r\n" +
			"Host: icap-server.net\r\n" +
			"Encapsulated: req-hdr=0, null-body=170\r\n\r\n" +
			"GET / HTTP/1.1\r\n" +
			"Host: www.origin-server.com\r\n" +
			"Accept: text/html, text/plain\r\n" +
			"Accept-Encoding: compress\r\n" +
			"Cookie: ff39fk3jur@4ii0e02i\r\n" +
			"If-None-Match: \"xyzzy\", \"r2d2xxxx\"\r\n\r\n")
	r := bufio.NewReader(buf)
	req, err := ReadRequest(r)

	if err != nil {
		t.Fatalf("Error while decoding request: %v", err)
	}

	checkString("Method", req.Method, "REQMOD", t)
	checkString("Protocol", req.Proto, "ICAP/1.0", t)
	checkString("Scheme", req.URL.Scheme, "icap", t)
	checkString("Host", req.URL.Host, "icap-server.net", t)
	checkString("Path", req.URL.Path, "/server", t)
	checkString("Query", req.URL.RawQuery, "arg=87", t)
	checkString("Host header", req.Header.Get("host"), "icap-server.net", t)
	checkString("Encapsulated header", req.Header.Get("encapsulated"), "req-hdr=0, null-body=170", t)
	checkString("Request method", req.Request.Method, "GET", t)
	checkString("Request host", req.Request.Host, "www.origin-server.com", t)
	checkString("Request Accept-Encoding header", req.Request.Header.Get("Accept-Encoding"), "compress", t)
}

func TestParserRESPMOD(t *testing.T) {
	buf := strings.NewReader(
		"RESPMOD icap://icap.example.org/satisf ICAP/1.0\r\n" +
			"Host: icap.example.org\r\n" +
			"Encapsulated: req-hdr=0, res-hdr=137, res-body=296\r\n\r\n" +
			"GET /origin-resource HTTP/1.1\r\n" +
			"Host: www.origin-server.com\r\n" +
			"Accept: text/html, text/plain, image/gif\r\n" +
			"Accept-Encoding: gzip, compress\r\n\r\n" +
			"HTTP/1.1 200 OK\r\n" +
			"Date: Mon, 10 Jan 2000 09:52:22 GMT\r\n" +
			"Server: Apache/1.3.6 (Unix)\r\n" +
			"ETag: \"63840-1ab7-378d415b\"\r\n" +
			"Content-Type: text/html\r\n" +
			"Content-Length: 51\r\n\r\n" +
			"33\r\n" +
			"This is data that was returned by an origin server.\r\n" +
			"0\r\n\r\n")
	r := bufio.NewReader(buf)
	req, err := ReadRequest(r)

	if err != nil {
		t.Fatalf("Error while decoding request: %v", err)
	}

	checkString("Request host", req.Request.Host, "www.origin-server.com", t)
	checkString("Response Server header", req.Response.Header.Get("Server"), "Apache/1.3.6 (Unix)", t)

	body, err := ioutil.ReadAll(req.Response.Body)
	if err != nil {
		t.Fatalf("Error while reading response body: %v", err)
	}
	checkString("Response body", string(body), "This is data that was returned by an origin server.", t)
}
