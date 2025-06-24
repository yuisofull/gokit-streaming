# Go Kit Streaming

This repository provides bidirectional streaming support for [Go Kit](https://github.com/go-kit/kit), a toolkit for
microservices in Go.

## Overview

Go Kit Streaming extends Go Kit with support for bidirectional streaming RPC using gRPC. It allows you to build
streaming services using the same patterns and abstractions that Go Kit provides for request-response services
by creating a new Endpoint type specifically for streaming.

```go
type StreamingEndpoint func(ctx context.Context, request <-chan interface{}) (response <-chan interface{}, Err func() error, err error)
```

It provides a channel-based API that allows you to handle streaming requests and responses in a way that is idiomatic to
Go's concurrency model.

## Features

- Bidirectional streaming support for gRPC
- Channel-based API for easy integration with Go's concurrency model

## Installation

```bash
go get github.com/yuisofull/gokitstreaming

```

## Usage

See the [examples](https://github.com/yuisofull/gokitstreaming/tree/main/examples/simple)