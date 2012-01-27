/*
An example of how to use go-icap.

Run this program and Squid on the same machine.
Put the following lines in squid.conf:

icap_enable on
icap_service service_req reqmod_precache icap://127.0.0.1:11344/golang
adaptation_access service_req allow all

(The ICAP server needs to be started before Squid is.)

Set your browser to use the Squid proxy.

Try browsing to http://gateway/ and http://java.com/
*/
package main

import (
	"code.google.com/p/go-icap"
	"fmt"
	"net/http"
	"os"
)

var ISTag = "\"GOLANG\""

func main() {
	// Set the files to be made available under http://gateway/
	http.Handle("/", http.FileServer(http.Dir(os.Getenv("HOME")+"/Sites")))

	icap.HandleFunc("/golang", toGolang)
	icap.ListenAndServe(":11344", icap.HandlerFunc(toGolang))
}

func toGolang(w icap.ResponseWriter, req *icap.Request) {
	h := w.Header()
	h.Set("ISTag", ISTag)
	h.Set("Service", "Golang redirector")

	switch req.Method {
	case "OPTIONS":
		h.Set("Methods", "REQMOD")
		h.Set("Allow", "204")
		w.WriteHeader(200, nil, false)
	case "REQMOD":
		switch req.Request.Host {
		case "gateway":
			// Run a fake HTTP server called gateway.
			icap.ServeLocally(w, req)
		case "java.com", "www.java.com":
			// Redirect the user to a more interesting language.
			req.Request.Host = "golang.org"
			req.Request.URL.Host = "golang.org"
			w.WriteHeader(200, req.Request, false)
			// TODO: copy the body (if any) from the original request.
		default:
			// Return the request unmodified.
			w.WriteHeader(204, nil, false)
		}
	default:
		w.WriteHeader(405, nil, false)
		fmt.Println("Invalid request method")
	}
}
