package unfurl

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnfurlCopiesInputToOutput(t *testing.T) {
	src := "one\nwrapped paragraph\n\n```go\nfmt.Println(\"preserved\")\n```\n"
	var out bytes.Buffer

	if err := Unfurl(strings.NewReader(src), &out); err != nil {
		t.Fatalf("Unfurl returned error: %v", err)
	}
	if got := out.String(); got != src {
		t.Fatalf("output mismatch:\nwant %q\n got %q", src, got)
	}
}

func TestUnfurlBytesReturnsFreshPassThroughCopy(t *testing.T) {
	src := []byte("alpha\nbeta\n")

	got, err := UnfurlBytes(src)
	if err != nil {
		t.Fatalf("UnfurlBytes returned error: %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("output mismatch:\nwant %q\n got %q", src, got)
	}

	got[0] = 'A'
	if src[0] != 'a' {
		t.Fatal("UnfurlBytes returned a slice aliasing the input")
	}
}
