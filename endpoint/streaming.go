package endpoint

import (
	"context"
)

// StreamingEndpoint represents a streaming endpoint that accepts a channel of requests
// and returns a channel of responses. If some thing goes wrong, such as request or
// response channel closing, they should check the Err function.
type StreamingEndpoint func(ctx context.Context, request <-chan interface{}) (response <-chan interface{}, Err func() error, err error)
