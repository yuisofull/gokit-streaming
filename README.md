# Go Kit Streaming

This repository provides bidirectional streaming support for [Go Kit](https://github.com/go-kit/kit), a toolkit for microservices in Go.

## Overview

Go Kit Streaming extends Go Kit with support for bidirectional streaming RPC using gRPC. It allows you to build streaming services using the same patterns and abstractions that Go Kit provides for request-response services.

## Features

- Bidirectional streaming support for gRPC
- Channel-based API for easy integration with Go's concurrency model
- Support for middleware, error handling, and other Go Kit features
- Compatible with standard Go Kit patterns and abstractions

## Installation

```bash
go get github.com/yuisofull/gokit-streaming
```

## Usage

### Server-side

```go
import (
    "github.com/yuisofull/gokit-streaming/transport/grpc"
)

// Define your service endpoint
func makeStreamingEndpoint() grpc.StreamingEndpoint {
    return func(ctx context.Context, requests <-chan interface{}) (<-chan grpc.StreamMessage, error) {
        responses := make(chan grpc.StreamMessage)
        
        go func() {
            defer close(responses)
            for request := range requests {
                // Process request and send response
                response := processRequest(request)
                responses <- grpc.StreamMessage{Message: response}
            }
        }()
        
        return responses, nil
    }
}

// Create a streaming server
server := grpc.NewStreamingServer(
    makeStreamingEndpoint(),
    decodeRequest,
    encodeResponse,
    pb.YourRequestType{},
)
```

### Client-side

```go
import (
    "github.com/yuisofull/gokit-streaming/transport/grpc"
)

// Create a streaming client
client := grpc.NewStreamingClient(
    conn,
    "/your.service/YourMethod",
    encodeRequest,
    decodeResponse,
    pb.YourResponseType{},
    serviceDesc,
)

// Get the streaming endpoint
endpoint := client.StreamingEndpoint()

// Create a channel for requests
requests := make(chan interface{})

// Call the endpoint
responses, err := endpoint(ctx, requests)
if err != nil {
    // Handle error
}

// Send requests and receive responses
go func() {
    defer close(requests)
    // Send requests
    requests <- yourRequest
}()

// Process responses
for response := range responses {
    if response.Err != nil {
        // Handle error
        continue
    }
    // Process response
    processResponse(response.Message)
}
```

## License

MIT

## Acknowledgements

This project is based on the work of the Go Kit team and community. It was created to provide streaming support that is not currently available in the main Go Kit repository.