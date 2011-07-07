package icap

import (
	"testing"
)

func TestStatusCodes(t *testing.T) {
	checkString("Message", StatusText(100), "Continue", t)
	checkString("Message", StatusText(401), "Unauthorized", t)
	checkString("Status-not-found message", StatusText(12345), "", t)
}
