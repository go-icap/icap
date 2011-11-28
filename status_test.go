// Copyright 2011 Andy Balholm. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icap

import (
	"testing"
)

func TestStatusCodes(t *testing.T) {
	checkString("Message", StatusText(100), "Continue", t)
	checkString("Message", StatusText(401), "Unauthorized", t)
	checkString("Status-not-found message", StatusText(12345), "", t)
}
