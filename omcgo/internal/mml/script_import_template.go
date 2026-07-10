package mml

import _ "embed"

// scriptImportTemplate is the single server-owned template exposed to all
// three web skins. Keeping it embedded prevents a deployment's working
// directory from changing the import contract.
//
//go:embed assets/MMLTemplate.txt
var scriptImportTemplate []byte
