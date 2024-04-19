package service

import (
	"fmt"

	"github.com/bnb-chain/blob-hub/models"
)

// Verify Interface Compliance
var _ error = (*Err)(nil)

// Err defines service errors.
type Err struct {
	Code    int64  `json:"code"`
	Message string `json:"error"`
}

func (e Err) Enrich(message string) Err {
	return Err{
		Code:    e.Code,
		Message: fmt.Sprintf("%s: %s", e.Message, message),
	}
}

func (e Err) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func InternalErrorWithError(err error) *models.Error {
	return &models.Error{
		Code:    500,
		Message: err.Error(),
	}
}

func BadRequestWithError(err error) *models.Error {
	return &models.Error{
		Code:    400,
		Message: err.Error(),
	}
}
