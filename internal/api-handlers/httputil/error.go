package httputil

import "errors"

var (
	ErrParseBodyDataFailed     = errors.New("failed to parse data")
	ErrContextDeadlineExceeded = errors.New("request context deadline exceeded")
)
