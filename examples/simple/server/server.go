package main

import (
	"context"
	"fmt"
	"github.com/yuisofull/gokitstreaming/endpoint"
	"github.com/yuisofull/gokitstreaming/examples/simple/pb"
	"github.com/yuisofull/gokitstreaming/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"os"

	"github.com/go-kit/log"
	transportgrpc "github.com/yuisofull/gokitstreaming/transport/grpc"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.Caller(5))

	server := transportgrpc.NewStreamingServer(
		streamingEndpoint(logger),
		decodeRequest,
		encodeResponse,
		pb.Request{},
		transportgrpc.StreamingServerErrorHandler(NewErrorHandler(logger)),
	)

	stream := NewStreamingServer(server)

	baseServer := grpc.NewServer()
	pb.RegisterSimpleServiceServer(baseServer, stream)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	if err := baseServer.Serve(listener); err != nil {
		logger.Log("err", err)
	}
}

// Request represents a request message
type Request struct {
	Message string
}

// Response represents a response message
type Response struct {
	Message string
}

// echoService is a private service struct that provides an echo functionality for processing and returning input data.
// It embeds a logger to enable structured logging for its operations.
type echoService struct {
	logger log.Logger
}

// Echo returns the input request as the response. It implements a simple echo functionality.
func (s *echoService) Echo(ctx context.Context, req interface{}) (interface{}, error) {
	return &Response{
		Message: req.(*Request).Message,
	}, nil
}

// NewEchoService creates and returns a new instance of echoService with the provided logger.
func NewEchoService(logger log.Logger) *echoService {
	return &echoService{
		logger: logger,
	}
}

// streamingServer is a gRPC server that implements the pb.SimpleServiceServer interface.
type streamingServer struct {
	stream transportgrpc.StreamingHandler
	pb.UnimplementedSimpleServiceServer
}

// StreamingMethod implements the gRPC method for pb.SimpleServiceServer
func (s *streamingServer) StreamingMethod(stream pb.SimpleService_StreamingMethodServer) error {
	_, err := s.stream.ServeGRPCStream(stream.Context(), stream)
	return err
}

// NewStreamingServer creates a new streamingServer with the provided StreamingHandler.
func NewStreamingServer(stream transportgrpc.StreamingHandler) *streamingServer {
	return &streamingServer{
		stream: stream,
	}
}

// streamingEndpoint returns a StreamingEndpoint that processes incoming requests, echoes them, and handles errors and limits.
func streamingEndpoint(logger log.Logger) endpoint.StreamingEndpoint {
	return func(ctx context.Context, req <-chan interface{}) (<-chan interface{}, func() error, error) {
		respCh := make(chan interface{}, 1)

		var streamErr error
		go func() {
			defer close(respCh)
			for i := 0; i < 3; i++ {
				r, ok := <-req
				// The stream can be closed by the client, so we check if the channel is still open.
				if !ok {
					return
				}
				msg, ok := r.(*Request)
				if !ok {
					streamErr = status.Errorf(codes.InvalidArgument, "expected *Request, got %T", r)
					return
				}

				svc := NewEchoService(logger)
				res, err := svc.Echo(ctx, msg)
				if err != nil {
					streamErr = status.Errorf(codes.Internal, "error processing request: %v", err)
					return
				}
				respCh <- res
			}
			streamErr = status.New(codes.ResourceExhausted, "Request limit exceeded.").Err()
		}()

		return respCh, func() error {
			return streamErr
		}, nil
	}
}

// decodeRequest decodes a gRPC request into the internal request format.
func decodeRequest(_ context.Context, req interface{}) (interface{}, error) {
	r, ok := req.(*pb.Request)
	if !ok {
		return nil, fmt.Errorf("expected *pb.Request, got %T", req)
	}
	return &Request{
		Message: r.Message,
	}, nil
}

// encodeResponse encodes the internal response into the gRPC response format.
func encodeResponse(_ context.Context, resp interface{}) (interface{}, error) {
	r, ok := resp.(*Response)
	if !ok {
		return nil, fmt.Errorf("expected *Data, got %T", resp)
	}
	return &pb.Response{
		Message: r.Message,
	}, nil
}

func NewErrorHandler(logger log.Logger) transport.ErrorHandler {
	return transport.ErrorHandlerFunc(func(ctx context.Context, err error) {
		logger.Log("err", err)
	})
}
