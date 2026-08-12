package main

import (
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

type transferEventSubscriber interface {
	Subscribe(subject string, handler event.EventHandler) (event.Subscription, error)
}

func subscribeTransferPolicyInvalidation(
	subscriber transferEventSubscriber,
	policy interface{ InvalidateCache() },
	logger *zap.Logger,
) (event.Subscription, error) {
	return subscriber.Subscribe(
		event.SubjectSysConfigSaved,
		transfercfg.HandleSysConfigSavedEvent(policy, logger),
	)
}
