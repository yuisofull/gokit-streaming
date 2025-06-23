package endpoint

import (
	"context"
)

// StreamingEndpoint represents a streaming endpoint that accepts a channel of requests
// and returns a channel of responses. This allows for bidirectional streaming communication.
type StreamingEndpoint func(ctx context.Context, request <-chan interface{}) (response <-chan struct {
	Data interface{}
	Err  error
}, err error)
