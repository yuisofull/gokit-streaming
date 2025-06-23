# Simple Streaming Example

This directory contains a simple example of how to use the Go Kit Streaming library for bidirectional streaming with gRPC.

## Overview

The example demonstrates how to:

1. Define a service with a streaming method
2. Implement the service
3. Create a gRPC server that uses the streaming functionality
4. Create a gRPC client that uses the streaming functionality

## Files

- `pb/proto.proto`: The protobuf definition for the service
- `server/server.go`: The server implementation
- `client/client.go`: The client implementation
- `pb/gen-proto.sh`: Script to generate protobuf code

## Running the Example

To run this example:

1. Generate the protobuf code:
   ```bash
   cd pb
   ./gen-proto.sh
   ```

2. Run the server:
   ```bash
   cd server
   go run server.go
   ```

3. In another terminal, run the client:
   ```bash
   cd client
   go run client.go
   ```

## Understanding the Example

### Server

The server implementation demonstrates:

- How to define a service with a streaming method
- How to implement the service
- How to create a streaming server
- How to handle requests and send responses
- How to return errors

The server processes up to 3 requests and then returns a "Request limit exceeded" error.

### Client

The client implementation demonstrates:

- How to create a streaming client
- How to get a streaming endpoint
- How to send requests
- How to receive and process responses
- How to handle errors

The client sends 5 requests and processes the responses, including handling any errors.

## Next Steps

To modify this example for your own use:

1. Update the proto.proto file with your service definition
2. Generate the protobuf code
3. Modify the server and client implementations to match your service
4. Run the server and client