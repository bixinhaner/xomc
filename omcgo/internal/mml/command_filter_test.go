package mml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newSupportedSet(resolved bool, paths ...string) *SupportedSet {
	m := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		m[p] = struct{}{}
	}
	return &SupportedSet{ProductResolved: resolved, Paths: m}
}

func TestAnnotateCommand(t *testing.T) {
	supported := newSupportedSet(true,
		"Device.X.A",
		"Device.X.B",
		"Device.Y.Parent.{i}.",
		"Device.Y.Parent.Child1",
		"Device.Y.Parent.Child2",
	)
	orphan := newSupportedSet(false)

	tests := []struct {
		name            string
		opType          string
		targetPathsJSON string
		targetObject    string
		supported       *SupportedSet
		wantVisible     bool
		wantSupported   int
		wantUnsupported []string
		wantProductFlag *bool
	}{
		{
			name:            "LST partial supported (≥1 → visible)",
			opType:          "LST",
			targetPathsJSON: `["Device.X.A","Device.X.NOT","Device.X.B"]`,
			supported:       supported,
			wantVisible:     true,
			wantSupported:   2, // 3 paths - 1 unsupported = 2 supported
			wantUnsupported: []string{"Device.X.NOT"},
			wantProductFlag: ptrBool(true),
		},
		{
			name:            "MOD all supported (visible, no unsupported)",
			opType:          "MOD",
			targetPathsJSON: `["Device.X.A","Device.X.B"]`,
			supported:       supported,
			wantVisible:     true,
			wantSupported:   2,
			wantUnsupported: []string{},
			wantProductFlag: ptrBool(true),
		},
		{
			name:            "LST none supported (hidden)",
			opType:          "LST",
			targetPathsJSON: `["Device.NOPE.A","Device.NOPE.B"]`,
			supported:       supported,
			wantVisible:     false,
			wantSupported:   0, // 0 in set
			wantUnsupported: []string{"Device.NOPE.A", "Device.NOPE.B"},
			wantProductFlag: ptrBool(true),
		},
		{
			name:            "ADD explicit-object-hit (visible)",
			opType:          "ADD",
			targetObject:    "Device.Y.Parent.",
			supported:       supported,
			wantVisible:     true,
			wantSupported:   1,
			wantUnsupported: nil,
			wantProductFlag: ptrBool(true),
		},
		{
			name:            "ADD child-prefix-only (hidden)",
			opType:          "ADD",
			targetObject:    "Device.Y.Other.",
			supported:       newSupportedSet(true, "Device.Y.Other.Child"),
			wantVisible:     false,
			wantSupported:   0,
			wantUnsupported: []string{"Device.Y.Other."},
			wantProductFlag: ptrBool(true),
		},
		{
			name:            "RMV prefix-miss (hidden)",
			opType:          "RMV",
			targetObject:    "Device.Z.Unknown.",
			supported:       supported,
			wantVisible:     false,
			wantSupported:   0, // no prefix match → 0 supported
			wantUnsupported: []string{"Device.Z.Unknown."},
			wantProductFlag: ptrBool(true),
		},
		{
			name:            "Orphan + LST: visible, all unsupported, productResolved=false",
			opType:          "LST",
			targetPathsJSON: `["Device.X.A","Device.X.B"]`,
			supported:       orphan,
			wantVisible:     true,
			wantSupported:   0, // 孤儿 supported set 空 → 0
			wantUnsupported: []string{"Device.X.A", "Device.X.B"},
			wantProductFlag: ptrBool(false),
		},
		{
			name:            "Orphan + ADD: visible, target_object as unsupported",
			opType:          "ADD",
			targetObject:    "Device.X.Y.",
			supported:       orphan,
			wantVisible:     true,
			wantSupported:   0, // 孤儿 → 0
			wantUnsupported: []string{"Device.X.Y."},
			wantProductFlag: ptrBool(false),
		},
		{
			name:            "Nil supported = no filtering, visible without annotation",
			opType:          "LST",
			targetPathsJSON: `["Device.X.A"]`,
			supported:       nil,
			wantVisible:     true,
			wantSupported:   0,
			wantUnsupported: nil,
			wantProductFlag: nil,
		},
		{
			name:            "Unknown op_type → visible (defensive)",
			opType:          "XYZ",
			supported:       supported,
			wantVisible:     true,
			wantProductFlag: ptrBool(true),
		},
		{
			name:            "LST with empty target_paths jsonb (parser returned 0 paths)",
			opType:          "LST",
			targetPathsJSON: `[]`,
			supported:       supported,
			wantVisible:     false, // 0 supported / 0 total → hide (defensive)
			wantSupported:   0,
			wantUnsupported: []string{},
			wantProductFlag: ptrBool(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnnotateCommand(tt.opType, []byte(tt.targetPathsJSON), tt.targetObject, tt.supported)
			assert.Equal(t, tt.wantVisible, got.Visible, "Visible")
			if tt.supported != nil {
				assert.Equal(t, tt.wantSupported, got.SupportedPathCount, "SupportedPathCount")
				if tt.wantUnsupported == nil {
					assert.Empty(t, got.UnsupportedPaths, "UnsupportedPaths should be empty")
				} else {
					assert.Equal(t, tt.wantUnsupported, got.UnsupportedPaths, "UnsupportedPaths")
				}
				if tt.wantProductFlag != nil {
					assert.Equal(t, *tt.wantProductFlag, got.ProductResolved, "ProductResolved")
				}
			}
		})
	}
}

func ptrBool(b bool) *bool { return &b }
