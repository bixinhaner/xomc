package mml

import _ "embed"

// Script import templates are server-owned and exposed to all three web skins.
// Keeping them embedded prevents a deployment's working directory from changing
// the import contract.
//
//go:embed assets/MMLTemplate.txt
var scriptImportTemplateZH []byte

//go:embed assets/MMLTemplate.en-US.txt
var scriptImportTemplateEN []byte

func scriptImportTemplateForLocale(locale string) []byte {
	if locale == "en-US" {
		return scriptImportTemplateEN
	}
	return scriptImportTemplateZH
}
