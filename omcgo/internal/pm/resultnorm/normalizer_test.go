package resultnorm

import (
	"errors"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name          string
		value         float64
		metadata      *Metadata
		numberProcess string
		want          float64
	}{
		{
			name:          "number defaults to half-up",
			value:         12.5,
			metadata:      &Metadata{Unit: "number", StatisType: "sum"},
			numberProcess: "",
			want:          13,
		},
		{
			name:          "number intUp rounds toward ceiling",
			value:         12.1,
			metadata:      &Metadata{Unit: "number", StatisType: "sum"},
			numberProcess: NumberProcessIntUp,
			want:          13,
		},
		{
			name:          "number intDown rounds toward floor",
			value:         12.9,
			metadata:      &Metadata{Unit: "number", StatisType: "sum"},
			numberProcess: NumberProcessIntDown,
			want:          12,
		},
		{
			name:          "number unknown process falls back to half-up",
			value:         12.5,
			metadata:      &Metadata{Unit: "number", StatisType: "sum"},
			numberProcess: "unexpected",
			want:          13,
		},
		{
			name:          "number process takes precedence over avg statis",
			value:         12.1,
			metadata:      &Metadata{Unit: "number", StatisType: "avg"},
			numberProcess: NumberProcessIntUp,
			want:          13,
		},
		{
			name:          "number none keeps non-integer to two decimals",
			value:         12.345,
			metadata:      &Metadata{Unit: "number", StatisType: "sum"},
			numberProcess: NumberProcessNone,
			want:          12.35,
		},
		{
			name:          "pct statis keeps two decimals",
			value:         98.765,
			metadata:      &Metadata{Unit: "ppm", StatisType: "pct"},
			numberProcess: NumberProcessIntUp,
			want:          98.77,
		},
		{
			name:          "avg statis keeps two decimals",
			value:         1.234,
			metadata:      &Metadata{Unit: "ms", StatisType: "avg"},
			numberProcess: NumberProcessIntDown,
			want:          1.23,
		},
		{
			name:          "percent unit caps above 100 after rounding",
			value:         100.006,
			metadata:      &Metadata{Unit: "%", StatisType: "pct"},
			numberProcess: DefaultNumberProcess,
			want:          100,
		},
		{
			name:          "non-percent pct is not capped",
			value:         100.006,
			metadata:      &Metadata{Unit: "ppm", StatisType: "pct"},
			numberProcess: DefaultNumberProcess,
			want:          100.01,
		},
		{
			name:          "ordinary integer remains integer",
			value:         42,
			metadata:      &Metadata{Unit: "s", StatisType: "sum"},
			numberProcess: DefaultNumberProcess,
			want:          42,
		},
		{
			name:          "ordinary non-integer keeps two decimals",
			value:         42.345,
			metadata:      &Metadata{Unit: "MByte", StatisType: "sum"},
			numberProcess: DefaultNumberProcess,
			want:          42.35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.value, tt.metadata, tt.numberProcess)
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Normalize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalize_MissingMetadataFails(t *testing.T) {
	tests := []struct {
		name     string
		metadata *Metadata
	}{
		{name: "nil metadata", metadata: nil},
		{name: "missing unit", metadata: &Metadata{StatisType: "pct"}},
		{name: "missing statis type", metadata: &Metadata{Unit: "%"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Normalize(1.23, tt.metadata, DefaultNumberProcess)
			if !errors.Is(err, ErrMissingMetadata) {
				t.Fatalf("Normalize() error = %v, want ErrMissingMetadata", err)
			}
		})
	}
}
