package icap

import (
	"testing"
	"bytes"
	"bufio"
)

func checkString(description, is, shouldBe string, t *testing.T) {
	if is != shouldBe {
		t.Fatalf("%s is %s (should be %s)", description, is, shouldBe)
	}
}

func TestHeaderParser(t *testing.T) {
	buf := bytes.NewBufferString(
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
}
