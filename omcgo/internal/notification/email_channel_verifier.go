package notification

import (
	"context"
	"fmt"
)

type EmailChannelVerifier struct{ sender *EmailSender }

func NewEmailChannelVerifier(sender *EmailSender) *EmailChannelVerifier {
	return &EmailChannelVerifier{sender: sender}
}

func (v *EmailChannelVerifier) Verify(ctx context.Context, config ChannelConfig) error {
	if v == nil || v.sender == nil {
		return ErrChannelVerificationUnavailable
	}
	if config.Channel != TemplateChannelEmail {
		return fmt.Errorf("verify notification email channel: unsupported channel %q", config.Channel)
	}
	if err := v.sender.Verify(ctx); err != nil {
		return fmt.Errorf("verify notification email channel: %w", err)
	}
	return nil
}
