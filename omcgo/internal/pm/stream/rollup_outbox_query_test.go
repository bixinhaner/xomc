package stream

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCounterRollupPeriodSelectRestrictsSourceVersions(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	query, args, err := counterRollupPeriodSelect(
		[]uuid.UUID{first, second},
		GranularityHourly,
		time.Date(2026, 7, 29, 19, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 29, 20, 0, 0, 0, time.UTC),
	).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "task_version_id IN") {
		t.Fatalf("period replay query lacks source-version predicate: %s", query)
	}
	joinedArgs := fmt.Sprint(args)
	if !strings.Contains(joinedArgs, first.String()) ||
		!strings.Contains(joinedArgs, second.String()) {
		t.Fatalf("period replay args lack source versions: %v", args)
	}
}
