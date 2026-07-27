package handbookasset

import _ "embed"

// The embedded archive is a build-time documentation template. The running OMC
// filters and repackages it against its actual Gin routes during startup.
//
//go:embed omc-api-handbook.tar.gz
var archive []byte

func Archive() []byte {
	return archive
}
