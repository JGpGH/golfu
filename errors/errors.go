package errors

import "errors"

var (
	ErrNotFound     = errors.New("element not found")
	ErrInvalidIndex = errors.New("invalid index")
)
