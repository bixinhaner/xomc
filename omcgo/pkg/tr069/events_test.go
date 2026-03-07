package tr069

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBootstrap(t *testing.T) {
	events := []EventStruct{
		{EventCode: EventBootstrap, CommandKey: ""},
	}
	assert.True(t, IsBootstrap(events))
	assert.False(t, IsPeriodic(events))
}

func TestIsPeriodic(t *testing.T) {
	events := []EventStruct{
		{EventCode: EventPeriodic, CommandKey: ""},
	}
	assert.False(t, IsBootstrap(events))
	assert.True(t, IsPeriodic(events))
}

func TestHasEvent(t *testing.T) {
	events := []EventStruct{
		{EventCode: EventBootstrap},
		{EventCode: EventValueChange},
	}
	assert.True(t, HasEvent(events, EventBootstrap))
	assert.True(t, HasEvent(events, EventValueChange))
	assert.False(t, HasEvent(events, EventPeriodic))
}

func TestEventCodes(t *testing.T) {
	events := []EventStruct{
		{EventCode: EventBootstrap},
		{EventCode: EventBoot},
	}
	codes := EventCodes(events)
	assert.Equal(t, []string{EventBootstrap, EventBoot}, codes)
}
