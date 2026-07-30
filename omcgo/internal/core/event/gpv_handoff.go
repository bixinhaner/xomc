package event

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type GPVHandoffConfig struct {
	Subject              string
	TargetDurable        string
	SourceConsumer       string
	ProvisionPullDurable string
	AckWait              time.Duration
	MaxDeliver           int
	MaxAckPending        int
}

type GPVHandoffResult struct {
	SourceConsumer string
	StartSequence  uint64
	Created        bool
}

func PrepareGPVHandoff(
	ctx context.Context,
	js nats.JetStreamContext,
	config GPVHandoffConfig,
) (GPVHandoffResult, error) {
	if js == nil {
		return GPVHandoffResult{}, fmt.Errorf("prepare GPV handoff: JetStream context is required")
	}
	if config.Subject == "" {
		return GPVHandoffResult{}, fmt.Errorf("prepare GPV handoff: subject is required")
	}
	if config.TargetDurable == "" {
		return GPVHandoffResult{}, fmt.Errorf("prepare GPV handoff: target durable is required")
	}
	if config.AckWait <= 0 {
		config.AckWait = gpvPullAckWait
	}
	if config.MaxDeliver <= 0 {
		config.MaxDeliver = maxDeliveries
	}
	if config.MaxAckPending <= 0 {
		config.MaxAckPending = gpvQueueMaxAckPending
	}

	stream, err := js.StreamNameBySubject(config.Subject, nats.Context(ctx))
	if err != nil {
		return GPVHandoffResult{}, fmt.Errorf(
			"resolve GPV stream for %s: %w",
			config.Subject,
			err,
		)
	}
	target, err := js.ConsumerInfo(stream, config.TargetDurable, nats.Context(ctx))
	if err == nil {
		if validateErr := validateGPVHandoffTarget(target, config); validateErr != nil {
			return GPVHandoffResult{}, validateErr
		}
		return GPVHandoffResult{
			StartSequence: target.Config.OptStartSeq,
			Created:       false,
		}, nil
	}
	if !errors.Is(err, nats.ErrConsumerNotFound) {
		return GPVHandoffResult{}, fmt.Errorf(
			"load GPV target consumer %s/%s: %w",
			stream,
			config.TargetDurable,
			err,
		)
	}

	source, err := findGPVHandoffSource(ctx, js, stream, config)
	if err != nil {
		return GPVHandoffResult{}, err
	}
	result := GPVHandoffResult{Created: true}
	deliverPolicy := nats.DeliverNewPolicy
	startSequence := uint64(0)
	if source != nil {
		result.SourceConsumer = source.Name
		startSequence = source.AckFloor.Stream + 1
		result.StartSequence = startSequence
		deliverPolicy = nats.DeliverByStartSequencePolicy
	}

	consumerConfig := &nats.ConsumerConfig{
		Durable:        config.TargetDurable,
		DeliverSubject: gpvHandoffDeliverSubject(config.TargetDurable),
		DeliverGroup:   config.TargetDurable,
		DeliverPolicy:  deliverPolicy,
		OptStartSeq:    startSequence,
		AckPolicy:      nats.AckExplicitPolicy,
		AckWait:        config.AckWait,
		MaxDeliver:     config.MaxDeliver,
		MaxAckPending:  config.MaxAckPending,
		FilterSubject:  config.Subject,
		ReplayPolicy:   nats.ReplayInstantPolicy,
	}
	created, err := js.AddConsumer(stream, consumerConfig, nats.Context(ctx))
	if err != nil {
		return GPVHandoffResult{}, fmt.Errorf(
			"precreate GPV target consumer %s/%s: %w",
			stream,
			config.TargetDurable,
			err,
		)
	}
	if err := validateGPVHandoffTarget(created, config); err != nil {
		return GPVHandoffResult{}, err
	}
	return result, nil
}

func findGPVHandoffSource(
	ctx context.Context,
	js nats.JetStreamContext,
	stream string,
	config GPVHandoffConfig,
) (*nats.ConsumerInfo, error) {
	if config.SourceConsumer != "" {
		source, err := js.ConsumerInfo(stream, config.SourceConsumer, nats.Context(ctx))
		if err != nil {
			return nil, fmt.Errorf(
				"load configured GPV source consumer %s/%s: %w",
				stream,
				config.SourceConsumer,
				err,
			)
		}
		if !consumerFiltersSubject(source.Config, config.Subject) {
			return nil, fmt.Errorf(
				"configured GPV source consumer %s/%s does not filter %s",
				stream,
				config.SourceConsumer,
				config.Subject,
			)
		}
		return source, nil
	}

	var candidates []*nats.ConsumerInfo
	for info := range js.Consumers(stream, nats.Context(ctx)) {
		if info == nil || info.Name == config.TargetDurable ||
			info.Name == config.ProvisionPullDurable {
			continue
		}
		if info.Config.DeliverSubject == "" ||
			!consumerFiltersSubject(info.Config, config.Subject) {
			continue
		}
		candidates = append(candidates, info)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list GPV source consumers for %s: %w", stream, err)
	}
	switch len(candidates) {
	case 0:
		return nil, nil
	case 1:
		return candidates[0], nil
	default:
		names := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			names = append(names, candidate.Name)
		}
		return nil, fmt.Errorf(
			"multiple GPV push consumers match %s: %s; configure the source consumer explicitly",
			config.Subject,
			strings.Join(names, ","),
		)
	}
}

func validateGPVHandoffTarget(info *nats.ConsumerInfo, config GPVHandoffConfig) error {
	if info == nil {
		return fmt.Errorf("GPV target consumer %s was not created", config.TargetDurable)
	}
	if info.Name != config.TargetDurable {
		return fmt.Errorf(
			"GPV target consumer name mismatch: got %s want %s",
			info.Name,
			config.TargetDurable,
		)
	}
	if !consumerFiltersSubject(info.Config, config.Subject) {
		return fmt.Errorf(
			"GPV target consumer %s does not filter %s",
			config.TargetDurable,
			config.Subject,
		)
	}
	if info.Config.DeliverSubject == "" ||
		info.Config.DeliverGroup != config.TargetDurable ||
		info.Config.AckPolicy != nats.AckExplicitPolicy {
		return fmt.Errorf(
			"GPV target consumer %s is incompatible with keyed queue binding",
			config.TargetDurable,
		)
	}
	return nil
}

func consumerFiltersSubject(config nats.ConsumerConfig, subject string) bool {
	if config.FilterSubject == subject {
		return true
	}
	for _, filter := range config.FilterSubjects {
		if filter == subject {
			return true
		}
	}
	return false
}

func gpvHandoffDeliverSubject(durable string) string {
	var builder strings.Builder
	builder.WriteString("_OMC.GPV.HANDOFF.")
	for _, char := range durable {
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '-',
			char == '_':
			builder.WriteRune(char)
		default:
			builder.WriteByte('_')
		}
	}
	return builder.String()
}
