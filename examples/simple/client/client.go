package main

import (
	"context"
	"fmt"
	"github.com/yuisofull/gokitstreaming/examples/simple/pb"
	transportgrpc "github.com/yuisofull/gokitstreaming/transport/grpc"
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

	endpoint := client.StreamingEndpoint()

	requests := make(chan interface{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call the endpoint
	responses, errFunc, err := endpoint(ctx, requests)
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

	for {
		resp, ok := <-responses
		// If the stream is aborted or closed, we check if there is an error
		if !ok {
			if err := errFunc(); err != nil {
				logger.Log("err", err)
			}
			return
		}
		response, ok := resp.(*Response)
		if !ok {
			logger.Log("err", fmt.Errorf("expected *Response, got %T", resp))
			return
		}
		logger.Log("response", response.Message)
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
		return nil, fmt.Errorf("expected *pb.Data, got %T", resp)
	}

	return &Response{
		Message: r.Message,
	}, nil
}
