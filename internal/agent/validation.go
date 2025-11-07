package agent

import (
	"context"

	"github.com/charmbracelet/crush/internal/result"
)

// AgentError represents agent-specific error types with proper error interface
type AgentError struct {
	Type    string
	Message string
}

func (e AgentError) Error() string {
	return e.Type + ": " + e.Message
}

// Agent error type constants
const (
	ErrorEmptyPrompt    = "EMPTY_PROMPT"
	ErrorSessionMissing = "SESSION_MISSING"  
	ErrorSessionBusy    = "SESSION_BUSY"
)

// NewAgentError creates a new AgentError
func NewAgentError(errorType, message string) *AgentError {
	return &AgentError{
		Type:    errorType,
		Message: message,
	}
}

// ValidateSessionCall validates session agent call parameters with type safety
func ValidateSessionCall(call SessionAgentCall) *AgentError {
	if call.Prompt == "" {
		return NewAgentError(ErrorEmptyPrompt, "Prompt cannot be empty")
	}
	if call.SessionID == "" {
		return NewAgentError(ErrorSessionMissing, "Session ID cannot be empty")
	}
	
	return nil // Return nil if validation passes
}

// PrepareSessionCall validates and prepares a session call with comprehensive checks
func PrepareSessionCall(call SessionAgentCall) result.Result[SessionAgentCall, *AgentError] {
	// Basic validation
	if agentErr := ValidateSessionCall(call); agentErr != nil {
		return result.Err[SessionAgentCall, *AgentError](agentErr)
	}
	
	// Additional preparation logic
	// TODO: Add more comprehensive validation logic
	// - Check model availability
	// - Validate token limits  
	// - Check permissions
	// - Verify attachments
	// - Add preparation timestamp if SessionAgentCall had Metadata field
	
	return result.Ok[SessionAgentCall, *AgentError](call)
}

// ExecuteWithRetry executes an agent operation with retry logic using Result types
func ExecuteWithRetry[T any](
	ctx context.Context,
	operation func() result.Result[T, *AgentError],
	maxRetries int,
) result.Result[T, *AgentError] {
	var lastErr *AgentError
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		operationResult := operation()
		if operationResult.IsSuccess() {
			return operationResult
		}
		
		lastErr = operationResult.Error()
		
		// TODO: Add exponential backoff
		// TODO: Add circuit breaker pattern
		// TODO: Add retry condition checking
		
		// Don't retry on validation errors
		switch lastErr.Type {
		case ErrorEmptyPrompt, ErrorSessionMissing:
			return result.Err[T, *AgentError](lastErr) // Don't retry validation errors
		}
		
		// Check context
		if ctx.Err() != nil {
			return result.Err[T, *AgentError](NewAgentError(ErrorSessionMissing, "Context cancelled"))
		}
	}
	
	return result.Err[T, *AgentError](lastErr)
}