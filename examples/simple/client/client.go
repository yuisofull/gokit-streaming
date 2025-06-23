package main

import (
	"context"
	"fmt"
	"github.com/yuisofull/gokit-streaming/examples/simple/pb"
	transportgrpc "github.com/yuisofull/gokit-streaming/transport/grpc"
	"google.golang.org/grpc"
	"os"
	"time"

	"github.com/go-kit/log"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	// Create a gRPC connection
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Create a streaming client
	client := transportgrpc.NewStreamingClient(
		conn,
		"/simple.SimpleService/StreamingMethod",
		encodeRequest,
		decodeResponse,
		pb.Response{},
		pb.SimpleService_ServiceDesc,
	)

	//Get the streaming endpoint
	endpoint := client.StreamingEndpoint()

	// Create a channel for requests
	requests := make(chan interface{})

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call the endpoint
	responses, err := endpoint(ctx, requests)
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	waitc := make(chan struct{})
	// Send requests
	go func() {
		defer close(requests)
		for i := 0; i < 5; i++ {
			select {
			case <-waitc:
				return
			default:
				requests <- &Request{Message: fmt.Sprintf("Request %d", i)}
			}
		}
	}()

	// Process responses
	for response := range responses {
		if response.Err != nil {
			logger.Log("err", response.Err)
			close(waitc)
			continue
		}
		resp := response.Message.(*Response)
		logger.Log("msg", "Received response", "response", resp.Message)
	}
}

type Request struct {
	Message string
}

type Response struct {
	Message string
}

func encodeRequest(_ context.Context, req interface{}) (interface{}, error) {
	r, ok := req.(*Request)
	if !ok {
		return nil, fmt.Errorf("expected *Request, got %T", req)
	}
	return &pb.Request{
		Message: r.Message,
	}, nil
}

func decodeResponse(_ context.Context, resp interface{}) (interface{}, error) {
	r, ok := resp.(*pb.Response)
	if !ok {
		return nil, fmt.Errorf("expected *pb.Response, got %T", resp)
	}

	return &Response{
		Message: r.Message,
	}, nil
}
