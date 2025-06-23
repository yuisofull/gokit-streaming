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
go get github.com/yuisofull/gokitstreaming
```

## Usage

See the [examples](https://github.com/yuisofull/gokitstreaming/tree/main/examples/simple)