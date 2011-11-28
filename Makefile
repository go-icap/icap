include $(GOROOT)/src/Make.inc

TARG=icap

GOFILES= \
	request.go \
	response.go \
	status.go \
	server.go \
	mux.go \
	bridge.go \
	
include $(GOROOT)/src/Make.pkg

format:
	gofmt -w ${GOFILES} *_test.go
