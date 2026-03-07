package xmlutil

import (
	"encoding/xml"
	"fmt"
	"io"
)

// FindElement advances the decoder until it finds a start element with the given local name.
func FindElement(decoder *xml.Decoder, name string) (*xml.StartElement, error) {
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil, fmt.Errorf("element %q not found", name)
		}
		if err != nil {
			return nil, err
		}

		if se, ok := token.(xml.StartElement); ok {
			if se.Name.Local == name {
				return &se, nil
			}
		}
	}
}

// ReadText reads the character data content of the current element.
func ReadText(decoder *xml.Decoder) (string, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch t := token.(type) {
		case xml.CharData:
			return string(t), nil
		case xml.EndElement:
			return "", nil
		}
	}
}

// SkipElement skips over the current element and all its children.
func SkipElement(decoder *xml.Decoder) error {
	return decoder.Skip()
}
