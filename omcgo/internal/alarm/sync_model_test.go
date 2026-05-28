package alarm

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestComputeDiff_SeparatesSameIdentifierByAdditionalInformation(t *testing.T) {
	makeAlarm := func(identifier, qualifier string) *model.Alarm {
		return &model.Alarm{
			AlarmIdentifier: identifier,
			Description:     "Cell unavailable",
			Severity:        model.AlarmMajor,
			AdditionalInfo:  map[string]string{"additional_information": qualifier},
		}
	}

	remote := []*model.Alarm{
		makeAlarm("11184", "cell=1"),
		makeAlarm("11184", "cell=2"),
	}
	local := []*model.Alarm{
		makeAlarm("11184", "cell=1"),
	}

	diff := ComputeDiff(remote, local)
	assert.Len(t, diff.ToAdd, 1)
	assert.Equal(t, "cell=2", diff.ToAdd[0].AdditionalInfo["additional_information"])
	assert.Len(t, diff.ToUpdate, 0)
	assert.Len(t, diff.ToClear, 0)
}