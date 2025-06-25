// Package grpc provides Go kit transport for gRPC, including streaming support.
// This file contains streaming-specific implementations for both client and server sides.

package grpc

import (
	"context"
	"github.com/yuisofull/gokitstreaming/endpoint"
	"google.golang.org/grpc/metadata"
	"io"
	"reflect"
	"sync"

	"github.com/yuisofull/gokitstreaming/transport"
	"google.golang.org/grpc"
)

type contextKey int

const (
	// ContextKeyRequestMethod is the context key used to store the request method.
	ContextKeyRequestMethod contextKey = iota
)

type StreamMessage struct {
	Message interface{}
	Err     error
}

// EncodeStreamRequestFunc encodes a single request object into a gRPC stream message.
// It's used by streaming clients to encode requests before sending them over the stream.
type EncodeStreamRequestFunc func(context.Context, interface{}) (interface{}, error)

// DecodeStreamResponseFunc decodes a single gRPC stream message into a response object.
// It's used by streaming clients to decode responses received from the stream.
type DecodeStreamResponseFunc func(context.Context, interface{}) (interface{}, error)

// ClientStreamRequestFunc is executed on the gRPC client stream before sending requests.
// It can be used to modify the outgoing metadata or perform other pre-request operations.
type ClientStreamRequestFunc func(context.Context, *metadata.MD) context.Context

// ClientStreamHeaderResponseFunc is executed when the client receives stream headers.
// It allows processing of metadata sent by the server in the stream headers.
type ClientStreamHeaderResponseFunc func(ctx context.Context, header *metadata.MD) context.Context

// ClientStreamTrailerResponseFunc is executed when the client receives stream trailers.
// It allows processing of metadata sent by the server in the stream trailers at the end of the stream.
type ClientStreamTrailerResponseFunc func(ctx context.Context, trailer *metadata.MD) context.Context

// ClientStreamFinalizerFunc is executed at the end of a streaming client operation.
// It's primarily intended for cleanup operations and error logging.
type ClientStreamFinalizerFunc func(ctx context.Context, err error)

// StreamingClient wraps a gRPC connection and provides streaming functionality.
// It handles the complexities of gRPC streaming while providing a channel-based interface.
type StreamingClient struct {
	client      *grpc.ClientConn                  // The underlying gRPC connection
	method      string                            // The gRPC method name
	enc         EncodeStreamRequestFunc           // Function to encode requests
	dec         DecodeStreamResponseFunc          // Function to decode responses
	grpcResp    reflect.Type                      // Type of the gRPC response message
	serviceDesc grpc.ServiceDesc                  // Service descriptor for the gRPC service
	before      []ClientStreamRequestFunc         // Functions executed before sending requests
	onHeader    []ClientStreamHeaderResponseFunc  // Functions executed when headers are received
	onTrailer   []ClientStreamTrailerResponseFunc // Functions executed when trailers are received
	finalizer   []ClientStreamFinalizerFunc       // Functions executed at the end of the stream
}

// NewStreamingClient constructs a new streaming client for a gRPC streaming method.
// It requires a gRPC connection, method name, encoding/decoding functions, and a service descriptor.
// The grpcResp parameter should be a zero-value instance of the expected response type.
func NewStreamingClient(
	cc *grpc.ClientConn,
	method string,
	enc EncodeStreamRequestFunc,
	dec DecodeStreamResponseFunc,
	grpcResp interface{},
	serviceDesc grpc.ServiceDesc,
	options ...StreamingClientOption,
) *StreamingClient {
	sc := &StreamingClient{
		client: cc,
		method: method,
		enc:    enc,
		dec:    dec,
		grpcResp: reflect.TypeOf(
			reflect.Indirect(
				reflect.ValueOf(grpcResp),
			).Interface(),
		),
		serviceDesc: serviceDesc,
	}
	for _, option := range options {
		option(sc)
	}
	return sc
}

// StreamingClientOption configures a StreamingClient with optional parameters.
type StreamingClientOption func(*StreamingClient)

// StreamingClientBefore sets functions that are executed before sending stream requests.
// These functions can modify the outgoing metadata or perform other pre-request operations.
func StreamingClientBefore(before ...ClientStreamRequestFunc) StreamingClientOption {
	return func(c *StreamingClient) { c.before = append(c.before, before...) }
}

// StreamingClientOnHeader sets functions that are executed when stream headers are received.
// These functions allow processing of metadata sent by the server in the stream headers.
func StreamingClientOnHeader(onHeader ...ClientStreamHeaderResponseFunc) StreamingClientOption {
	return func(c *StreamingClient) { c.onHeader = append(c.onHeader, onHeader...) }
}

// StreamingClientOnTrailer sets functions that are executed when stream trailers are received.
// These functions allow processing of metadata sent by the server in the stream trailers.
func StreamingClientOnTrailer(onTrailer ...ClientStreamTrailerResponseFunc) StreamingClientOption {
	return func(c *StreamingClient) { c.onTrailer = append(c.onTrailer, onTrailer...) }
}

// StreamingEndpoint returns a StreamingEndpoint that handles the gRPC streaming communication.
// The returned endpoint accepts a channel of requests and returns a channel of responses,
// along with a function to handle error when the response channel is closed, if any.
func (c *StreamingClient) StreamingEndpoint() endpoint.StreamingEndpoint {
	return func(ctx context.Context, reqCh <-chan interface{}) (<-chan interface{}, func() error, error) {
		var (
			streamErr error
			once      sync.Once
			mu        sync.Mutex
		)

		setErr := func(err error) {
			once.Do(func() {
				mu.Lock()
				defer mu.Unlock()
				streamErr = err
			})
		}

		getErr := func() error {
			mu.Lock()
			defer mu.Unlock()
			return streamErr
		}

		// Add method name to context for middleware usage
		ctx = context.WithValue(ctx, ContextKeyRequestMethod, c.method)

		// Apply before functions to modify outgoing metadata
		md := &metadata.MD{}
		for _, f := range c.before {
			ctx = f(ctx, md)
		}

		ctx = metadata.NewOutgoingContext(ctx, *md)

		stream, err := c.client.NewStream(ctx, &c.serviceDesc.Streams[0], c.method)
		if err != nil {
			return nil, nil, err
		}

		respCh := make(chan interface{})

		// Sending goroutine
		go func() {
			defer stream.CloseSend()

			for req := range reqCh {
				msg, err := c.enc(ctx, req)
				if err != nil {
					setErr(err)
					return
				}
				if err := stream.SendMsg(msg); err != nil {
					if err == io.EOF {
						return
					}
					setErr(err)
					return
				}
			}
		}()

		// Receiving goroutine
		go func() {
			defer func() {
				close(respCh)
				if c.finalizer != nil {
					for _, f := range c.finalizer {
						f(ctx, getErr())
					}
				}
			}()

			// Handle stream headers
			header, err := stream.Header()
			if err != nil {
				setErr(err)
				return
			}
			for _, f := range c.onHeader {
				ctx = f(ctx, &header)
			}

			// Receive loop
			for {
				msgPtr := reflect.New(c.grpcResp).Interface()
				if err := stream.RecvMsg(msgPtr); err != nil {
					trailer := stream.Trailer()
					for _, f := range c.onTrailer {
						ctx = f(ctx, &trailer)
					}
					if err == io.EOF {
						return
					}
					setErr(err)
					return
				}

				decoded, err := c.dec(ctx, msgPtr)
				if err != nil {
					setErr(err)
					return
				}

				respCh <- decoded
			}
		}()

		return respCh, func() error {
			return getErr()
		}, nil
	}
}

// DecodeStreamRequestFunc decodes a single gRPC stream message into a request object.
// It's used by streaming servers to decode requests received from the stream.
type DecodeStreamRequestFunc func(context.Context, interface{}) (interface{}, error)

// EncodeStreamResponseFunc encodes a single response object into a gRPC stream message.
// It's used by streaming servers to encode responses before sending them over the stream.
type EncodeStreamResponseFunc func(context.Context, interface{}) (interface{}, error)

// ServerStreamRequestFunc is executed on the gRPC server stream before processing requests.
// It can be used to extract information from metadata or perform other pre-processing operations.
type ServerStreamRequestFunc func(context.Context, metadata.MD) context.Context

// ServerStreamResponseFunc is executed on the gRPC server stream before sending responses.
// It allows modification of response headers and trailers.
type ServerStreamResponseFunc func(ctx context.Context, header *metadata.MD, trailer *metadata.MD) context.Context

// ServerStreamFinalizerFunc can be used to perform work at the end of gRPC
// streaming, after the stream is aborted.
type ServerStreamFinalizerFunc func(ctx context.Context, err error)

// StreamingServer wraps a StreamingEndpoint and implements streaming gRPC server functionality.
// It handles the complexities of gRPC server streaming while providing a channel-based interface.
type StreamingServer struct {
	endpoint     endpoint.StreamingEndpoint  // The business logic endpoint
	dec          DecodeStreamRequestFunc     // Function to decode requests
	enc          EncodeStreamResponseFunc    // Function to encode responses
	grpcReqType  reflect.Type                // Type of the gRPC request message
	before       []ServerStreamRequestFunc   // Functions executed before processing requests
	after        []ServerStreamResponseFunc  // Functions executed before sending responses
	errorHandler transport.ErrorHandler      // General error handler
	finalizer    []ServerStreamFinalizerFunc // finalizer defines functions to be executed at the end of gRPC streaming.
}

// StreamingHandler defines the interface for gRPC streaming handlers.
// It's implemented by StreamingServer to handle incoming streaming requests.
type StreamingHandler interface {
	ServeGRPCStream(ctx context.Context, stream grpc.ServerStream) (retctx context.Context, err error)
}

// NewStreamingServer constructs a new streaming server that wraps the provided endpoint.
// It requires a StreamingEndpoint, encoding/decoding functions, and a zero-value instance
// of the expected request type.
func NewStreamingServer(
	endpoint endpoint.StreamingEndpoint,
	dec DecodeStreamRequestFunc,
	enc EncodeStreamResponseFunc,
	grpcReq interface{},
	options ...StreamingServerOption,
) *StreamingServer {
	ss := &StreamingServer{
		endpoint: endpoint,
		dec:      dec,
		enc:      enc,
		grpcReqType: reflect.TypeOf(
			reflect.Indirect(
				reflect.ValueOf(grpcReq),
			).Interface(),
		),
		errorHandler: transport.NewNopErrorHandler(),
	}
	for _, option := range options {
		option(ss)
	}
	return ss
}

// StreamingServerOption configures a StreamingServer with optional parameters.
type StreamingServerOption func(*StreamingServer)

// StreamingServerBefore sets functions that are executed before processing stream requests.
// These functions can extract information from metadata or perform other pre-processing operations.
func StreamingServerBefore(before ...ServerStreamRequestFunc) StreamingServerOption {
	return func(s *StreamingServer) { s.before = append(s.before, before...) }
}

// StreamingServerAfter sets functions that are executed before sending stream responses.
// These functions allow modification of response headers and trailers.
func StreamingServerAfter(after ...ServerStreamResponseFunc) StreamingServerOption {
	return func(s *StreamingServer) { s.after = append(s.after, after...) }
}

// StreamingServerErrorHandler sets the error handler for non-terminal errors.
// By default, errors are logged to a no-op logger.
func StreamingServerErrorHandler(errorHandler transport.ErrorHandler) StreamingServerOption {
	return func(s *StreamingServer) { s.errorHandler = errorHandler }
}

// ServerStreamFinalizer is executed at the end of gRPC streaming.
// By default, no finalizer is registered.
func ServerStreamFinalizer(finalizer ...ServerStreamFinalizerFunc) StreamingServerOption {
	return func(s *StreamingServer) { s.finalizer = append(s.finalizer, finalizer...) }
}

// ServeGRPCStream implements the StreamingHandler interface.
// It handles incoming gRPC streams by converting them to channel-based communication
// and delegating to the configured StreamingEndpoint.
func (s *StreamingServer) ServeGRPCStream(ctx context.Context, stream grpc.ServerStream) (retctx context.Context, err error) {
	// Extract metadata from the incoming context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	if len(s.finalizer) > 0 {
		defer func() {
			for _, f := range s.finalizer {
				f(ctx, err)
			}
		}()
	}

	reqCh := make(chan interface{})

	errCh := make(chan error)

	// Goroutine to handle receiving requests from the stream
	go func() {
		defer close(reqCh)
		for {
			msgPtr := reflect.New(s.grpcReqType).Interface()
			if err := stream.RecvMsg(msgPtr); err != nil {
				if err == io.EOF {
					return // Stream closed normally
				}
				errCh <- err
				return
			}

			// Apply before functions to process metadata
			for _, f := range s.before {
				ctx = f(ctx, md)
			}
			decoded, err := s.dec(ctx, msgPtr)
			if err != nil {
				errCh <- err
				return
			}
			reqCh <- decoded
		}
	}()

	// Call the business logic endpoint
	respCh, errFunc, err := s.endpoint(ctx, reqCh)
	if err != nil {
		s.errorHandler.Handle(ctx, err)
		return ctx, err
	}

	// Apply after functions to set headers and trailers
	var mdHeader, mdTrailer metadata.MD
	for _, f := range s.after {
		ctx = f(ctx, &mdHeader, &mdTrailer)
	}
	if len(mdHeader) > 0 {
		if err = grpc.SendHeader(ctx, mdHeader); err != nil {
			s.errorHandler.Handle(ctx, err)
			return ctx, err
		}
	}

	if len(mdTrailer) > 0 {
		if err = grpc.SetTrailer(ctx, mdTrailer); err != nil {
			s.errorHandler.Handle(ctx, err)
			return ctx, err
		}
	}

	for {
		select {
		case err := <-errCh:
			s.errorHandler.Handle(ctx, err)
			return ctx, err
		case resp, ok := <-respCh:
			if !ok {
				if errFunc != nil {
					if err := errFunc(); err != nil {
						s.errorHandler.Handle(ctx, err)
						return ctx, err
					}
				}
				return ctx, nil
			}

			encoded, err := s.enc(ctx, resp)
			if err != nil {
				s.errorHandler.Handle(ctx, err)
				return ctx, err
			}
			if err := stream.SendMsg(encoded); err != nil {
				s.errorHandler.Handle(ctx, err)
				return ctx, err
			}
		}
	}
}
