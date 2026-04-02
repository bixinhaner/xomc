package device

import "github.com/google/uuid"

// BatchOperationResult summarises the outcome of a batch operation.
type BatchOperationResult struct {
	Total     int              `json:"total"`
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Errors    []BatchItemError `json:"errors,omitempty"`
}

// BatchItemError records the failure reason for a single item in a batch operation.
type BatchItemError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// BatchIDsRequest is the request body for batch operations that accept a list of device IDs.
type BatchIDsRequest struct {
	IDs []uuid.UUID `json:"ids" binding:"required"`
}
