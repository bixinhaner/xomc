package handbookasset

import _ "embed"

//go:embed omc-api-handbook.tar.gz
var archive []byte

func Archive() []byte {
	return archive
}
