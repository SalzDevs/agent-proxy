package groxy

import (
	"errors"
	"net/http"
)

// BlockError represents an intentional block response returned by a hook.
type BlockError struct {
	StatusCode int
	Message    string
}

func (e *BlockError) Error() string {
	return e.Message
}

// Block creates an error that tells Groxy to stop processing and return a
// response with statusCode and message to the client.
func Block(statusCode int, message string) error {
	return &BlockError{StatusCode: statusCode, Message: message}
}

func blockError(err error) (*BlockError, bool) {
	var block *BlockError
	if errors.As(err, &block) {
		return block, true
	}

	return nil, false
}

func writeBlock(w http.ResponseWriter, block *BlockError) {
	statusCode := block.StatusCode
	if statusCode < 100 || statusCode > 999 {
		statusCode = http.StatusForbidden
	}

	http.Error(w, block.Message, statusCode)
}
