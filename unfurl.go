package unfurl

import (
	"bytes"
	"fmt"
	"io"

	"github.com/michaelquigley/unfurl/internal/emit"
	"github.com/michaelquigley/unfurl/internal/tokenize"
)

// Unfurl reads markdown from r and writes the unfurled output to w.
// Soft line breaks inside paragraph content are collapsed; every other
// CommonMark/GFM construct is preserved byte-for-byte.
func Unfurl(r io.Reader, w io.Writer) error {
	doc, err := tokenize.Tokenize(r)
	if err != nil {
		return fmt.Errorf("tokenize markdown: %w", err)
	}
	blocks := tokenize.Group(doc)
	if err := emit.Emit(w, doc, blocks); err != nil {
		return fmt.Errorf("emit markdown: %w", err)
	}
	return nil
}

// UnfurlBytes is a convenience wrapper around Unfurl for in-memory use.
// The returned slice is a fresh allocation; the input is not modified.
func UnfurlBytes(src []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := Unfurl(bytes.NewReader(src), &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
