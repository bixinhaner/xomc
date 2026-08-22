package parammodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSHA256HexUsesFullXMLContent(t *testing.T) {
	require.Equal(t,
		"ba14a9e279e6cd8575084f3ff5c898b7fa58d8ee466b9a818e608a005cc6ff03",
		sha256Hex([]byte(`<parameterModel paramModel="BLQ"></parameterModel>`)))
	require.NotEqual(t,
		sha256Hex([]byte(`<parameterModel paramModel="BLQ"></parameterModel>`)),
		sha256Hex([]byte(`<parameterModel paramModel="BLQ"><parameters/></parameterModel>`)))
}

func TestShouldSkipUnchangedParamModelRequiresSameHashAndBuiltinRows(t *testing.T) {
	const (
		loadedFrom = "param-mappings/BLQ.xml"
		hash       = "ba14a9e279e6cd8575084f3ff5c898b7fa58d8ee466b9a818e608a005cc6ff03"
		otherHash  = "d05d3ea55ddc4252503335289582b396c3258381f273dd10de4da4f2180b988a"
	)

	require.True(t, shouldSkipUnchangedParamModel(loadedFrom, hash, loadedFrom, hash, 1))
	require.False(t, shouldSkipUnchangedParamModel(loadedFrom, otherHash, loadedFrom, hash, 1), "must not skip only because builtin rows exist")
	require.False(t, shouldSkipUnchangedParamModel(loadedFrom, hash, loadedFrom, hash, 0))
	require.False(t, shouldSkipUnchangedParamModel("param-mappings/MLN.xml", hash, loadedFrom, hash, 1))
}
