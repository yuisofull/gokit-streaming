package endpoint

import (
	"context"
)

// StreamingEndpoint represents a streaming endpoint that accepts a channel of requests
// and returns a channel of responses.If a client a closed channel, they should check
// the returned Err() function to see what happened.
type StreamingEndpoint func(ctx context.Context, request <-chan interface{}) (response <-chan interface{}, Err func() error, err error)
