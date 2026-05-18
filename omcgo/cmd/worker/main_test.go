package main

import (
	"reflect"
	"testing"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/stretchr/testify/require"
)

func TestWireUnknownAlarmFallback_InjectsBothReceivers(t *testing.T) {
	alarmReceiver := &alarm.AlarmReceiver{}
	expeditedReceiver := &alarm.ExpeditedEventReceiver{}
	alarmDefRegistry := &definition.Registry{}
	productRegistry := &product.Registry{}

	alarmReceiver, expeditedReceiver, resolver := wireUnknownAlarmFallback(
		alarmReceiver,
		expeditedReceiver,
		alarmDefRegistry,
		productRegistry,
	)

	require.NotNil(t, resolver)
	require.False(t, nilField(t, alarmReceiver, "alarmDefRegistry"))
	require.False(t, nilField(t, alarmReceiver, "productResolver"))
	require.False(t, nilField(t, expeditedReceiver, "alarmDefRegistry"))
	require.False(t, nilField(t, expeditedReceiver, "productResolver"))
}

func nilField(t *testing.T, target any, fieldName string) bool {
	t.Helper()
	value := reflect.ValueOf(target)
	require.Equal(t, reflect.Ptr, value.Kind())
	field := value.Elem().FieldByName(fieldName)
	require.Truef(t, field.IsValid(), "field %s should exist", fieldName)
	return field.IsNil()
}
