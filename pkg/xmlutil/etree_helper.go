package xmlutil

import (
	"strings"

	"github.com/beevik/etree"
)

// WalkParameterTree traverses an etree element tree, calling the callback
// for each leaf element with its full dotted path.
func WalkParameterTree(root *etree.Element, callback func(path string, elem *etree.Element) error) error {
	return walkTree(root, "", callback)
}

func walkTree(elem *etree.Element, prefix string, callback func(string, *etree.Element) error) error {
	path := prefix
	if path != "" {
		path += "."
	}
	path += elem.Tag

	if len(elem.ChildElements()) == 0 {
		return callback(path, elem)
	}

	for _, child := range elem.ChildElements() {
		if err := walkTree(child, path, callback); err != nil {
			return err
		}
	}
	return nil
}

// FindParameterByPath finds an element by its dotted path (e.g., "Device.ManagementServer.URL").
func FindParameterByPath(root *etree.Element, path string) *etree.Element {
	parts := strings.Split(path, ".")
	current := root

	for _, part := range parts {
		found := false
		for _, child := range current.ChildElements() {
			if child.Tag == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	return current
}
