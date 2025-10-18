# JSON-RPC Client Package

A lightweight Go package for making JSON-RPC calls over HTTP. This implementation provides a simple and efficient client for communicating with JSON-RPC servers. Based on https://github.com/ybbus/jsonrpc

## Overview

The package implements the JSON-RPC 2.0 specification with support for:

- **Single RPC calls**: Make individual method calls to a remote service
- **Batch calls**: Send multiple requests in a single HTTP request
- **Custom headers**: Add HTTP headers to requests (useful for authentication)
- **Flexible parameters**: Automatically normalize method parameters
- **Type conversions**: Helper methods to extract typed values from responses

## Core Components

### RPCClient Interface

The main entry point defining five methods:

- `Call()`: Make a single RPC call
- `CallRaw()`: Make a call with a pre-constructed request
- `CallFor()`: Make a call and automatically unmarshal the result
- `CallBatch()`: Send multiple requests in one batch
- `CallBatchRaw()`: Send a batch without modifying request IDs

### Request/Response Types

- `RPCRequest`: Represents a JSON-RPC request with method, params, and ID
- `RPCResponse`: Contains the result, error, and ID from the server
- `RPCError`: Represents JSON-RPC errors with code, message, and optional data

### Implementation Details

The `rpcClient` struct handles the low-level HTTP communication:

- Encodes requests as JSON
- Sends POST requests to the configured endpoint
- Decodes JSON responses using a decoder that supports number precision
- Propagates both HTTP and RPC errors appropriately

## Quick Start

### Installation

Clone the repository and navigate to the project directory:

```bash
git clone https://github.com/thornxyz/json_rpc_go.git
cd json_rpc_go
```

### Running the Example

The project includes simple `client.go` and `server.go` files demonstrating a complete working example.

Start the server in one terminal:

```bash
go run server.go
```

In another terminal, run the client:

```bash
go run client.go
```

The client will make RPC calls to the server and display the results.
