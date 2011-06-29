include $(GOROOT)/src/Make.inc

TARG=icap

GOFILES= \
	request.go \
	
include $(GOROOT)/src/Make.pkg

format:
	gofmt -w ${GOFILES} *_test.go
