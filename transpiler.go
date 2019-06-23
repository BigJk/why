package why

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// Transpile will convert a html document that contains
// script tags <!? ... ?!> into a fully working tengo script.
// Html will be wrapped into http.write("...") calls.
func Transpile(in io.Reader, out io.Writer) error {
	iteration := 0
	tags := [][]byte{[]byte("<!?"), []byte("?!>")}

	scanner := bufio.NewScanner(in)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, tags[iteration%2]); i >= 0 {
			return i + 3, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	writer := bufio.NewWriter(out)
	for scanner.Scan() {
		switch iteration % 2 {
		case 0:
			if _, err := writer.WriteString("; http.write(`"); err != nil {
				return err
			}
			if _, err := writer.Write(scanner.Bytes()); err != nil {
				return err
			}
			if _, err := writer.WriteString("`);"); err != nil {
				return err
			}
		case 1:
			if _, err := writer.Write(scanner.Bytes()); err != nil {
				return err
			}
		}
		iteration++
	}

	if iteration%2 == 0 {
		return errors.New("missing closing tags")
	}

	return writer.Flush()
}
