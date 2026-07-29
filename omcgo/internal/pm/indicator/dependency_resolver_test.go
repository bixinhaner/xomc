package indicator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveDependencyClosureExpandsKPIToCounters(t *testing.T) {
	arithmetic := map[string]string{
		"K900010076": "C000060216/C000060273*100",
		"K900010006": "(C000000012/C000000005)*(C000120002/C000120001)*(C000010080/C000010070)*100",
	}
	isCounter := map[string]bool{
		"K900010076": false,
		"K900010006": false,
		"C000060216": true,
		"C000060273": true,
		"C000000012": true,
		"C000000005": true,
		"C000120002": true,
		"C000120001": true,
		"C000010080": true,
		"C000010070": true,
	}

	got, err := ResolveDependencyClosure(
		[]string{"K900010076", "K900010006"},
		arithmetic,
		isCounter,
	)

	require.NoError(t, err)
	require.Equal(t, []string{"K900010006", "K900010076"}, got.KPIs)
	require.Equal(t, []string{
		"C000000005",
		"C000000012",
		"C000010070",
		"C000010080",
		"C000060216",
		"C000060273",
		"C000120001",
		"C000120002",
	}, got.Counters)
}

func TestResolveDependencyClosureRecursivelyExpandsNestedKPI(t *testing.T) {
	got, err := ResolveDependencyClosure(
		[]string{"K2"},
		map[string]string{
			"K2": "K1/C3",
			"K1": "C1+C2",
		},
		map[string]bool{
			"K2": false,
			"K1": false,
			"C1": true,
			"C2": true,
			"C3": true,
		},
	)

	require.NoError(t, err)
	require.Equal(t, []string{"K1", "K2"}, got.KPIs)
	require.Equal(t, []string{"C1", "C2", "C3"}, got.Counters)
}

func TestResolveDependencyClosureRejectsCircularKPI(t *testing.T) {
	_, err := ResolveDependencyClosure(
		[]string{"K1"},
		map[string]string{"K1": "K2/C1", "K2": "K1/C2"},
		map[string]bool{"K1": false, "K2": false, "C1": true, "C2": true},
	)

	require.ErrorIs(t, err, ErrCircularIndicatorDependency)
}

func TestResolveDependencyClosureRejectsUnknownIndicator(t *testing.T) {
	_, err := ResolveDependencyClosure(
		[]string{"K1"},
		map[string]string{"K1": "C1/C_MISSING"},
		map[string]bool{"K1": false, "C1": true},
	)

	require.ErrorIs(t, err, ErrUnknownIndicatorDependency)
	require.True(t, errors.Is(err, ErrUnknownIndicatorDependency))
	require.ErrorContains(t, err, "C_MISSING")
}
