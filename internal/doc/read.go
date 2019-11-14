package doc

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"regexp"
)

// Fragment is a parseable portion of a Document.
type Fragment []byte

// Document is a collection of related Fragments.
type Document struct {
	Parts []Fragment
}

var splitter = regexp.MustCompile("---[\t\f\r ]*\n")

func splitDocuments(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for _, m := range splitter.FindAllIndex(data, -1) {
		// The '---' must be anchored to the start of a line,
		// so we only accept matches that are at the beginning
		// of a buffer or follow a newline.
		if m[0] == 0 || data[m[0]-1] == '\n' {
			// Token is from the buffer start until the start of the separator.
			token := data[0:m[0]]
			// Advance over the separator.
			advance := m[1]

			log.Printf("advancing %d", advance)
			return advance, token, nil
		}
	}

	if atEOF {
		log.Printf("at eof")
		return len(data), bytes.TrimSuffix(data, []byte{'-', '-', '-'}), nil
	}

	log.Printf("need more")
	// Keep reading ...
	return 0, nil, nil
}

// ReadDocument reads a stream of Fragments that are separated by a
// YAML document separator (see https://yaml.org/spec/1.0/#id2561718).
// The contents of each Fragment is opaque and need not be YAML.
func ReadDocument(in io.Reader) (*Document, error) {
	doc := Document{}

	scanner := bufio.NewScanner(in)
	scanner.Split(splitDocuments)

	for scanner.Scan() {
		doc.Parts = append(doc.Parts, scanner.Bytes())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &doc, nil
}
