package license

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasFeature(t *testing.T) {
	featureList := FeatureList(`{
        "Dashboard": "All",
        "eNB": {
            "Monitor": ["Synchronize", "Settings"],
            "Maintenance": "All"
        },
        "gNB": ["Inventory"]
    }`)

	tests := []struct {
		name string
		path []string
		want bool
	}{
		{name: "top level all", path: []string{"Dashboard", "Widgets"}, want: true},
		{name: "nested item", path: []string{"eNB", "Monitor", "Settings"}, want: true},
		{name: "nested item missing", path: []string{"eNB", "Monitor", "Active"}, want: false},
		{name: "nested all", path: []string{"eNB", "Maintenance", "MML"}, want: true},
		{name: "flat list item", path: []string{"gNB", "Inventory"}, want: true},
		{name: "flat list missing", path: []string{"gNB", "Monitor"}, want: false},
		{name: "empty path", path: nil, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, HasFeature(featureList, test.path...))
		})
	}
}

func TestHasFeatureInvalidJSON(t *testing.T) {
	assert.False(t, HasFeature(FeatureList(`{invalid`), "Dashboard"))
}
