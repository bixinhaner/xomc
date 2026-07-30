package stream

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuiltinNetworkTaskID(t *testing.T) {
	tests := []struct {
		technology string
		want       string
		ok         bool
	}{
		{technology: "lte", want: "0184dddd-0001-4000-8000-000000000001", ok: true},
		{technology: " LTE ", want: "0184dddd-0001-4000-8000-000000000001", ok: true},
		{technology: "nr", want: "0184dddd-0001-4000-8000-000000000002", ok: true},
		{technology: "GSM", want: "0184dddd-0001-4000-8000-000000000003", ok: true},
		{technology: "unknown", want: uuid.Nil.String(), ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.technology, func(t *testing.T) {
			got, ok := BuiltinNetworkTaskID(tt.technology)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got.String())
		})
	}
}
