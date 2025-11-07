package errors

import (
	"fmt"
	"strings"
	"time"
)

// APIErrorType represents categorization of API errors with strong typing
type APIErrorType string

const (
	APIErrorTypeAPI                APIErrorType = "API_ERROR"
	APIErrorTypeAuthentication     APIErrorType = "AUTHENTICATION_ERROR"
	APIErrorTypeRateLimit        APIErrorType = "RATE_LIMIT_EXCEEDED"
	APIErrorTypeNetwork          APIErrorType = "NETWORK_ERROR"
	APIErrorTypeInvalidRequest   APIErrorType = "INVALID_REQUEST"
	APIErrorTypeBilling          APIErrorType = "BILLING_ERROR"
	APIErrorTypeModelUnavailable APIErrorType = "MODEL_UNAVAILABLE"
	APIErrorTypeContentPolicy    APIErrorType = "CONTENT_POLICY_VIOLATION"
	APIErrorTypeTokenLimit       APIErrorType = "TOKEN_LIMIT_EXCEEDED"
	APIErrorTypeServer          APIErrorType = "SERVER_ERROR"
	APIErrorTypeBadRequest      APIErrorType = "BAD_REQUEST"
	APIErrorTypeUnauthorized    APIErrorType = "UNAUTHORIZED"
	APIErrorTypeForbidden       APIErrorType = "FORBIDDEN"
	APIErrorTypeNotFound        APIErrorType = "NOT_FOUND"
	APIErrorTypeRateLimited    APIErrorType = "RATE_LIMITED"
	APIErrorTypeInternalServer APIErrorType = "INTERNAL_SERVER_ERROR"
	APIErrorTypeBadGateway     APIErrorType = "BAD_GATEWAY"
	APIErrorTypeServiceUnavailable APIErrorType = "SERVICE_UNAVAILABLE"
)

// StructuredAPIError represents a well-structured, type-safe API error
type StructuredAPIError struct {
	Type      APIErrorType `json:"type"`
	Title     string       `json:"title"`
	Message   string       `json:"message"`
	Details   string       `json:"details"`
	Timestamp string       `json:"timestamp"`
	IsNil     bool         `json:"is_nil"`
}

// NewAPIError creates a new StructuredAPIError with the given type and message
func NewAPIError(errorType APIErrorType, title, message, details string) *StructuredAPIError {
	return &StructuredAPIError{
		Type:      errorType,
		Title:     title,
		Message:   message,
		Details:   details,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		IsNil:     false,
	}
}

// NewNilAPIError creates a StructuredAPIError representing a nil error (should never happen in production)
func NewNilAPIError() *StructuredAPIError {
	return &StructuredAPIError{
		Type:      APIErrorTypeAPI,
		Title:     "Internal Error",
		Message:   "An unexpected condition occurred during error processing",
		Details:   "Error handling was triggered with no actual error - this indicates a system bug",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		IsNil:     true,
	}
}

// FormatAPIError converts a Go error to a type-safe StructuredAPIError
func FormatAPIError(err error) *StructuredAPIError {
	if err == nil {
		return NewNilAPIError()
	}
	
	errStr := err.Error()
	lowerErr := strings.ToLower(errStr)
	
	// Smart error classification with type-safe dispatch
	switch {
	case strings.Contains(lowerErr, "unauthorized") || strings.Contains(lowerErr, "authentication") || strings.Contains(lowerErr, "invalid api key"):
		return NewAPIError(APIErrorTypeAuthentication, "Authentication Error", errStr, buildAuthenticationDetails())
		
	case strings.Contains(lowerErr, "rate limit") || strings.Contains(lowerErr, "too many requests") || strings.Contains(lowerErr, "quota exceeded"):
		return NewAPIError(APIErrorTypeRateLimit, "Rate Limit Exceeded", errStr, buildRateLimitDetails())
		
	case strings.Contains(lowerErr, "timeout") || strings.Contains(lowerErr, "deadline exceeded"):
		return NewAPIError(APIErrorTypeNetwork, "Request Timeout", errStr, buildTimeoutDetails())
		
	case strings.Contains(lowerErr, "connection") || strings.Contains(lowerErr, "network"):
		return NewAPIError(APIErrorTypeNetwork, "Network Error", errStr, buildNetworkDetails())
		
	case strings.Contains(lowerErr, "invalid request") || strings.Contains(lowerErr, "bad request"):
		return NewAPIError(APIErrorTypeInvalidRequest, "Invalid Request", errStr, buildInvalidRequestDetails())
		
	case strings.Contains(lowerErr, "insufficient credits") || strings.Contains(lowerErr, "billing") || strings.Contains(lowerErr, "payment"):
		return NewAPIError(APIErrorTypeBilling, "Billing Error", errStr, buildBillingDetails())
		
	case strings.Contains(lowerErr, "model not found") || strings.Contains(lowerErr, "invalid model"):
		return NewAPIError(APIErrorTypeModelUnavailable, "Model Unavailable", errStr, buildModelUnavailableDetails())
		
	case strings.Contains(lowerErr, "content policy") || strings.Contains(lowerErr, "content filter") || strings.Contains(lowerErr, "safety"):
		return NewAPIError(APIErrorTypeContentPolicy, "Content Policy Violation", errStr, buildContentPolicyDetails())
		
	case strings.Contains(lowerErr, "token") && (strings.Contains(lowerErr, "exceeded") || strings.Contains(lowerErr, "limit")):
		return NewAPIError(APIErrorTypeTokenLimit, "Token Limit Exceeded", errStr, buildTokenLimitDetails())
		
	case strings.Contains(lowerErr, "server error") || strings.Contains(lowerErr, "internal error") || strings.Contains(lowerErr, "502") || strings.Contains(lowerErr, "503") || strings.Contains(lowerErr, "500"):
		return NewAPIError(APIErrorTypeServer, "Server Error", errStr, buildServerErrorDetails())
		
	default:
		// Fallback HTTP status code detection
		if strings.Contains(errStr, "400") {
			return NewAPIError(APIErrorTypeBadRequest, "Bad Request", errStr, buildHTTPDetails(400))
		} else if strings.Contains(errStr, "401") {
			return NewAPIError(APIErrorTypeUnauthorized, "Unauthorized", errStr, buildHTTPDetails(401))
		} else if strings.Contains(errStr, "403") {
			return NewAPIError(APIErrorTypeForbidden, "Forbidden", errStr, buildHTTPDetails(403))
		} else if strings.Contains(errStr, "404") {
			return NewAPIError(APIErrorTypeNotFound, "Not Found", errStr, buildHTTPDetails(404))
		} else if strings.Contains(errStr, "429") {
			return NewAPIError(APIErrorTypeRateLimited, "Rate Limited", errStr, buildHTTPDetails(429))
		} else if strings.Contains(errStr, "500") {
			return NewAPIError(APIErrorTypeInternalServer, "Internal Server Error", errStr, buildHTTPDetails(500))
		} else if strings.Contains(errStr, "502") {
			return NewAPIError(APIErrorTypeBadGateway, "Bad Gateway", errStr, buildHTTPDetails(502))
		} else if strings.Contains(errStr, "503") {
			return NewAPIError(APIErrorTypeServiceUnavailable, "Service Unavailable", errStr, buildHTTPDetails(503))
		}
		
		// Default case
		return NewAPIError(APIErrorTypeAPI, "API Error", errStr, "")
	}
}

// ToUIString converts StructuredAPIError to UI display format
func (e *StructuredAPIError) ToUIString() (string, string) {
	if e == nil {
		return "Internal Error", "A critical system error occurred"
	}
	return e.Title, e.Details
}

// Detail builders - centralized user guidance
func buildAuthenticationDetails() string {
	return `This appears to be an authentication issue. Please check:
• Your API key is valid and not expired
• The API key has sufficient permissions
• You're using the correct provider endpoint`
}

func buildRateLimitDetails() string {
	return `You've hit the rate limit. Options:
• Wait a moment before retrying
• Check your usage quota
• Consider upgrading your plan`
}

func buildTimeoutDetails() string {
	return `The request timed out. This could be due to:
• Slow model response time
• Network connectivity issues
• Server overload`
}

func buildNetworkDetails() string {
	return `Network connectivity issue detected:
• Check your internet connection
• Verify the API endpoint is accessible
• Try again in a few moments`
}

func buildInvalidRequestDetails() string {
	return `The API request was invalid:
• Check the request parameters
• Verify the model name exists
• Ensure request format is correct`
}

func buildBillingDetails() string {
	return `Billing or payment issue:
• Check your account balance
• Update your payment method
• Verify your subscription is active`
}

func buildModelUnavailableDetails() string {
	return `Model availability issue:
• Verify the model name is correct
• Check if the model is available in your region
• Try a different model`
}

func buildContentPolicyDetails() string {
	return `Content policy violation:
• Review and modify your prompt
• Avoid restricted content
• Check content guidelines`
}

func buildTokenLimitDetails() string {
	return `Token limit exceeded:
• Shorten your prompt
• Break into smaller requests
• Use a model with higher token limits`
}

func buildServerErrorDetails() string {
	return `Server-side error occurred:
• This is a temporary issue with the API
• Try again in a few minutes
• Check provider status page`
}

func buildHTTPDetails(statusCode int) string {
	return fmt.Sprintf(`HTTP %d error detected:
• Check the API documentation for this status code
• Verify your request parameters
• Try again in a few moments`, statusCode)
}