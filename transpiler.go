package why

import (
	"bytes"
	"errors"
)

func writeTag(buf *bytes.Buffer, data []byte) {
	buf.WriteString("; http.write(`")
	buf.Write(data)
	buf.WriteString("`);")
}

// Transpile will convert a html document that contains
// script tags <!? ... ?!> into a fully working tengo script.
// Html will be wrapped into http.write("...") calls.
func Transpile(data []byte) ([]byte, error) {
	final := &bytes.Buffer{}
	for {
		pos := bytes.Index(data, []byte("<!?"))
		if pos == -1 {
			break
		}

		if pos > 0 {
			writeTag(final, data[:pos])
		}

		endPos := bytes.Index(data, []byte("?!>"))
		if endPos == -1 {
			return nil, errors.New("missing closing tags")
		}

		final.Write(data[pos+3 : endPos])
		data = data[endPos+3:]
	}

	if len(data) > 0 {
		writeTag(final, data)
	}

	return final.Bytes(), nil
}
