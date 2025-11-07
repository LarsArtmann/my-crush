package agent

import (
	"context"
	"fmt"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/errors"
)

// MessageProcessor handles message preparation and transformation
type MessageProcessor struct {
	cfg *config.Config
	msg pubsub.Publisher[message.Message]
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(cfg *config.Config, msgPublisher pubsub.Publisher[message.Message]) *MessageProcessor {
	return &MessageProcessor{
		cfg: cfg,
		msg: msgPublisher,
	}
}

// ProcessedMessages contains prepared messages for agent execution
type ProcessedMessages struct {
	Messages       []message.Message
	HasContext    bool
	SystemMessage message.Message
}

// PrepareMessages processes incoming messages for agent execution
func (mp *MessageProcessor) PrepareMessages(ctx context.Context, msgs []message.Message) (*ProcessedMessages, error) {
	if len(msgs) == 0 {
		return nil, errors.ValidationError("no messages to process")
	}

	// Validate message sequence
	if err := mp.ValidateMessages(msgs); err != nil {
		return nil, err
	}
	
	return &ProcessedMessages{
		Messages:       msgs,
		HasContext:    len(msgs) > 1,
		SystemMessage: msgs[0], // First message is typically system
	}, nil
}

// SaveMessage saves a message to message service
func (mp *MessageProcessor) SaveMessage(ctx context.Context, msg message.Message) error {
	mp.msg.Publish(pubsub.CreatedEvent, msg)
	return nil
}

// ValidateMessages validates message sequence
func (mp *MessageProcessor) ValidateMessages(msgs []message.Message) error {
	if len(msgs) == 0 {
		return errors.ValidationError("message list cannot be empty")
	}

	// Check for valid role sequence
	for i, msg := range msgs {
		if err := msg.Role.Validate(); err != nil {
			return errors.ValidationErrorWithCause(fmt.Sprintf("invalid role at index %d", i), err)
		}
	}

	// Check for system message at beginning
	if len(msgs) > 0 && !msgs[0].Role.IsSystem() {
		return errors.ValidationError("first message should be system message")
	}

	return nil
}

// GetTokenCount estimates token count for messages
func (mp *MessageProcessor) GetTokenCount(msgs []message.Message) (int, error) {
	// Simple token estimation (rough approximation: ~4 characters per token)
	totalChars := 0
	for _, msg := range msgs {
		content := msg.Content()
		totalChars += len(content.Text)
	}
	
	// Rough estimation: 4 chars = 1 token
	estimatedTokens := totalChars / 4
	
	if estimatedTokens == 0 && totalChars > 0 {
		estimatedTokens = 1 // Minimum 1 token if there's content
	}
	
	return estimatedTokens, nil
}

// FormatMessages converts messages to prompt format
func (mp *MessageProcessor) FormatMessages(msgs []message.Message) string {
	var prompt string
	for _, msg := range msgs {
		content := msg.Content()
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, content.Text)
	}
	return prompt
}